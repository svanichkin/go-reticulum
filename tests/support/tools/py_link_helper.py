#!/usr/bin/env python3

import argparse
import os
import sys
import time

import RNS

APP_NAME = "paritylink"
ASPECT = "hold"
EXPECT_CLOSE = False


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


def apply_keepalive(link, keepalive_seconds):
    if keepalive_seconds is None or keepalive_seconds <= 0:
        return
    link.keepalive = keepalive_seconds
    link.stale_time = keepalive_seconds * RNS.Link.STALE_FACTOR


def await_path(destination_hash, timeout):
    if not RNS.Transport.has_path(destination_hash):
        RNS.Transport.request_path(destination_hash)
    deadline = time.time() + timeout
    while time.time() < deadline:
        if RNS.Transport.has_path(destination_hash):
            return True
        time.sleep(0.1)
    return RNS.Transport.has_path(destination_hash)


def run_listener(identity, wait_seconds, keepalive_seconds):
    done = {"closed": False}

    destination = RNS.Destination(identity, RNS.Destination.IN, RNS.Destination.SINGLE, APP_NAME, ASPECT)

    def on_link_established(link):
        apply_keepalive(link, keepalive_seconds)
        log("EVENT established " + RNS.hexrep(link.link_id, delimit=False))
        if link.get_mtu() is not None:
            log(f"EVENT mtu={link.get_mtu()} mdu={link.get_mdu()}")

        def on_identified(_link, remote_identity):
            log("EVENT identified " + RNS.hexrep(remote_identity.hash, delimit=False))

        def on_closed(link):
            log(f"EVENT closed reason={link.teardown_reason} initiator={link.initiator}")
            done["closed"] = True

        link.set_remote_identified_callback(on_identified)
        link.set_link_closed_callback(on_closed)

    destination.set_link_established_callback(on_link_established)
    log("LISTEN_HASH " + RNS.hexrep(destination.hash, delimit=False))
    time.sleep(1.0)
    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if done["closed"]:
            return 0
        destination.announce()
        for _ in range(30):
            if done["closed"]:
                return 0
            time.sleep(0.1)

    log("EVENT timeout")
    return 0


def run_client(identity, destination_hex, identify, teardown, hold_seconds, wait_seconds, keepalive_seconds):
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
    closed = {"seen": False}

    def on_closed(link):
        log(f"EVENT closed reason={link.teardown_reason} initiator={link.initiator}")
        closed["seen"] = True

    link.set_link_closed_callback(on_closed)

    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if link.status == RNS.Link.ACTIVE:
            break
        time.sleep(0.1)

    if link.status != RNS.Link.ACTIVE:
        print("link not active", file=sys.stderr)
        return 1

    apply_keepalive(link, keepalive_seconds)
    log("EVENT established " + RNS.hexrep(link.link_id, delimit=False))
    if link.get_mtu() is not None:
        log(f"EVENT mtu={link.get_mtu()} mdu={link.get_mdu()}")

    if identify:
        link.identify(identity)
        log("EVENT identify_sent")
        time.sleep(0.2)

    if hold_seconds > 0:
        time.sleep(hold_seconds)
        if EXPECT_CLOSE:
            if link.status == RNS.Link.CLOSED:
                log("EVENT stale_closed")
            else:
                print(f"link did not close during hold, status={link.status}", file=sys.stderr)
                return 1
        else:
            if link.status != RNS.Link.ACTIVE:
                print(f"link not active after hold, status={link.status}", file=sys.stderr)
                return 1
            log("EVENT still_active")

    if teardown:
        link.teardown()
        deadline = time.time() + wait_seconds
        while time.time() < deadline:
            if link.status == RNS.Link.CLOSED:
                break
            time.sleep(0.1)
        if link.status != RNS.Link.CLOSED:
            print("link did not close", file=sys.stderr)
            return 1

    return 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default=None)
    parser.add_argument("--identity", default=None)
    parser.add_argument("--destination", default=None)
    parser.add_argument("--listen", action="store_true")
    parser.add_argument("--identify", action=argparse.BooleanOptionalAction, default=True)
    parser.add_argument("--teardown", action="store_true")
    parser.add_argument("--expect-close", action="store_true")
    parser.add_argument("--trace", action="store_true")
    parser.add_argument("--hold-seconds", type=float, default=0)
    parser.add_argument("--wait-seconds", type=float, default=30)
    parser.add_argument("--keepalive-seconds", type=float, default=0)
    args = parser.parse_args()

    reticulum = RNS.Reticulum(configdir=args.config, loglevel=(RNS.LOG_DEBUG if args.trace else 2))
    _ = reticulum
    identity = prepare_identity(args.identity)
    global EXPECT_CLOSE
    EXPECT_CLOSE = args.expect_close

    if args.listen:
        raise SystemExit(run_listener(identity, args.wait_seconds, args.keepalive_seconds))

    if args.destination is None:
        print("destination is required in client mode", file=sys.stderr)
        raise SystemExit(1)

    raise SystemExit(
        run_client(
            identity,
            args.destination,
            args.identify,
            args.teardown,
            args.hold_seconds,
            args.wait_seconds,
            args.keepalive_seconds,
        )
    )


if __name__ == "__main__":
    main()
