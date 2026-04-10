#!/usr/bin/env python3

import argparse
import hashlib
import os
import sys
import time

import RNS

APP_NAME = "parityresource"
ASPECT = "blob"


def log(msg):
    print(msg, flush=True)


def prepare_identity(path):
    if path is None:
        path = RNS.Reticulum.identitypath + "/" + APP_NAME

    if os.path.isfile(path):
        identity = RNS.Identity.from_file(path)
        if identity is not None:
            return identity

    os.makedirs(os.path.dirname(path), exist_ok=True)
    identity = RNS.Identity()
    identity.to_file(path)
    return identity


def make_large_payload(size):
    if size < 1:
        size = 1
    pattern = b"GO-PY-RESOURCE-PARITY-"
    return (pattern * ((size // len(pattern)) + 1))[:size]


def make_cancel_payload(size):
    if size < 1:
        size = 1
    out = bytearray(size)
    x = 0x5D
    for i in range(size):
        x = ((x * 73) + 41) & 0xFF
        out[i] = x ^ (i & 0xFF)
    return bytes(out)


def await_path(destination_hash, timeout):
    if not RNS.Transport.has_path(destination_hash):
        RNS.Transport.request_path(destination_hash)
    deadline = time.time() + timeout
    while time.time() < deadline:
        if RNS.Transport.has_path(destination_hash):
            return True
        time.sleep(0.1)
    return RNS.Transport.has_path(destination_hash)


def run_listener(identity, wait_seconds, small_payload, large_size, mode, incompressible_large):
    expected_large = make_cancel_payload(large_size) if incompressible_large else make_large_payload(large_size)
    seen = {"small": False, "large": False}

    destination = RNS.Destination(identity, RNS.Destination.IN, RNS.Destination.SINGLE, APP_NAME, ASPECT)

    def on_link_established(link):
        link.set_resource_strategy(RNS.Link.ACCEPT_APP)

        def on_adv(resource):
            log(
                "EVENT adv transfer={} data={} parts={} segments={} compressed={}".format(
                    resource.get_transfer_size(),
                    resource.get_data_size(),
                    resource.get_parts(),
                    resource.get_segments(),
                    resource.is_compressed(),
                )
            )
            if mode == "reject":
                log("EVENT rejected kind=small")
                return False
            return True

        def on_started(resource):
            log(
                "EVENT started transfer={} data={} parts={} segments={} compressed={}".format(
                    resource.get_transfer_size(),
                    resource.get_data_size(),
                    resource.get_parts(),
                    resource.get_segments(),
                    resource.is_compressed(),
                )
            )
        def on_concluded(resource):
            if mode == "cancel" and resource.status == RNS.Resource.FAILED:
                log("EVENT canceled_by_initiator")
                seen["canceled"] = True
                return
            if resource.status != RNS.Resource.COMPLETE:
                log(f"EVENT resource_failed status={resource.status}")
                return

            with open(resource.data.name, "rb") as fh:
                data = fh.read()

            meta = resource.metadata if isinstance(resource.metadata, dict) else {}
            kind = meta.get("kind", "")
            digest = hashlib.sha256(data).hexdigest()
            log(f"EVENT concluded kind={kind} bytes={len(data)} sha256={digest}")

            if kind == "small" and data == small_payload:
                seen["small"] = True
            elif kind == "large" and data == expected_large:
                seen["large"] = True
            else:
                log(f"EVENT mismatch kind={kind}")

        link.set_resource_callback(on_adv)
        link.set_resource_started_callback(on_started)
        link.set_resource_concluded_callback(on_concluded)

    destination.set_link_established_callback(on_link_established)
    log("LISTEN_HASH " + RNS.hexrep(destination.hash, delimit=False))
    time.sleep(1.0)
    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if (mode == "cancel" and seen.get("canceled")) or (mode != "cancel" and seen["small"] and seen["large"]):
            return 0
        destination.announce()
        time.sleep(2.0)

    if mode == "cancel":
        if not seen.get("canceled"):
            print("cancel not observed", file=sys.stderr)
            return 1
        return 0
    if not seen["small"] or not seen["large"]:
        print(f"resources not fully received: small={seen['small']} large={seen['large']}", file=sys.stderr)
        return 1
    return 0


def run_client(identity, destination_hex, wait_seconds, small_payload, large_size, mode, incompressible_large):
    destination_hash = bytes.fromhex(destination_hex)
    if not await_path(destination_hash, wait_seconds):
        print("path not found", file=sys.stderr)
        return 1

    remote_identity = RNS.Identity.recall(destination_hash)
    if remote_identity is None:
        print("could not recall remote identity", file=sys.stderr)
        return 1

    destination = RNS.Destination(remote_identity, RNS.Destination.OUT, RNS.Destination.SINGLE, APP_NAME, ASPECT)
    link = RNS.Link(destination)
    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if link.status == RNS.Link.ACTIVE:
            break
        time.sleep(0.1)

    if link.status != RNS.Link.ACTIVE:
        print("link not active", file=sys.stderr)
        return 1

    link.identify(identity)
    time.sleep(1.0)

    if mode == "reject":
        status = send_resource_expect_failure(link, "small", small_payload, wait_seconds)
        if status is None:
            return 1
        log(f"EVENT {status}")
        link.teardown()
        return 0
    if mode == "cancel":
        cancel_size = large_size
        if cancel_size < (1 << 20):
            cancel_size = 1 << 20
        status = send_resource_and_cancel(link, "large", make_cancel_payload(cancel_size), wait_seconds)
        if status is None:
            return 1
        log(f"EVENT {status}")
        link.teardown()
        return 0

    if not send_resource(link, "small", small_payload, wait_seconds):
        return 1
    large_payload = make_cancel_payload(large_size) if incompressible_large else make_large_payload(large_size)
    if not send_resource(link, "large", large_payload, wait_seconds):
        return 1

    link.teardown()
    return 0


def send_resource(link, kind, payload, wait_seconds):
    done = {"resource": None}

    def on_done(resource):
        done["resource"] = resource

    resource = RNS.Resource(
        payload,
        link,
        metadata={"kind": kind},
        advertise=True,
        auto_compress=False,
        callback=on_done,
        timeout=wait_seconds,
    )
    _ = resource

    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if done["resource"] is not None:
            break
        time.sleep(0.1)

    if done["resource"] is None:
        print(f"{kind} resource timeout", file=sys.stderr)
        return False
    if done["resource"].status != RNS.Resource.COMPLETE:
        print(f"{kind} resource status={done['resource'].status}", file=sys.stderr)
        return False

    digest = hashlib.sha256(payload).hexdigest()
    log(
        "EVENT sent kind={} bytes={} sha256={} segments={} parts={} compressed={}".format(
            kind,
            len(payload),
            digest,
            done["resource"].get_segments(),
            done["resource"].get_parts(),
            done["resource"].is_compressed(),
        )
    )
    return True


def send_resource_expect_failure(link, kind, payload, wait_seconds):
    done = {"resource": None}

    def on_done(resource):
        done["resource"] = resource

    resource = RNS.Resource(
        payload,
        link,
        metadata={"kind": kind},
        advertise=True,
        auto_compress=False,
        callback=on_done,
        timeout=wait_seconds,
    )
    _ = resource

    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if done["resource"] is not None:
            break
        time.sleep(0.1)

    if done["resource"] is None:
        print(f"{kind} resource timeout", file=sys.stderr)
        return None
    if done["resource"].status == RNS.Resource.REJECTED:
        return "rejected"
    if done["resource"].status == RNS.Resource.FAILED:
        return "canceled"
    if done["resource"].status == RNS.Resource.TRANSFERRING:
        return "canceled"
    print(f"{kind} unexpected status={done['resource'].status}", file=sys.stderr)
    return None


def send_resource_and_cancel(link, kind, payload, wait_seconds):
    done = {"resource": None}

    def on_done(resource):
        done["resource"] = resource

    resource = RNS.Resource(
        payload,
        link,
        metadata={"kind": kind},
        advertise=True,
        auto_compress=False,
        callback=on_done,
        timeout=wait_seconds,
    )
    cancel_deadline = time.time() + min(wait_seconds, 5.0)
    while resource.status < RNS.Resource.TRANSFERRING and time.time() < cancel_deadline:
        time.sleep(0.05)
    resource.cancel()

    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if done["resource"] is not None:
            break
        time.sleep(0.1)

    if done["resource"] is None:
        print(f"{kind} resource timeout", file=sys.stderr)
        return None
    settle_deadline = time.time() + min(wait_seconds, 5.0)
    while done["resource"].status == RNS.Resource.TRANSFERRING and time.time() < settle_deadline:
        time.sleep(0.1)
    if done["resource"].status == RNS.Resource.FAILED:
        return "canceled"
    if done["resource"].status == RNS.Resource.TRANSFERRING:
        return "canceled"
    print(f"{kind} unexpected status={done['resource'].status}", file=sys.stderr)
    return None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default=None)
    parser.add_argument("--identity", default=None)
    parser.add_argument("--destination", default=None)
    parser.add_argument("--listen", action="store_true")
    parser.add_argument("--mode", default="normal")
    parser.add_argument("--trace", action="store_true")
    parser.add_argument("--incompressible-large", action="store_true")
    parser.add_argument("--wait-seconds", type=float, default=45)
    parser.add_argument("--small-payload", default="parity-small")
    parser.add_argument("--large-size", type=int, default=24576)
    args = parser.parse_args()

    reticulum = RNS.Reticulum(configdir=args.config, loglevel=(RNS.LOG_EXTREME if args.trace else 2))
    _ = reticulum
    identity = prepare_identity(args.identity)
    small_payload = args.small_payload.encode("utf-8")

    if args.listen:
        raise SystemExit(run_listener(identity, args.wait_seconds, small_payload, args.large_size, args.mode, args.incompressible_large))

    if args.destination is None:
        print("destination is required", file=sys.stderr)
        raise SystemExit(1)

    raise SystemExit(run_client(identity, args.destination, args.wait_seconds, small_payload, args.large_size, args.mode, args.incompressible_large))


if __name__ == "__main__":
    main()
