#!/usr/bin/env python3

import argparse
import os
import sys
import time
import re

import RNS

APP_NAME = "paritycache"
ASPECT = "replay"


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


def await_path(destination_hash, timeout):
    if not RNS.Transport.has_path(destination_hash):
        RNS.Transport.request_path(destination_hash)
    deadline = time.time() + timeout
    while time.time() < deadline:
        if RNS.Transport.has_path(destination_hash):
            return True
        time.sleep(0.1)
    return RNS.Transport.has_path(destination_hash)


def run_listener(identity, payload, wait_seconds):
    done = {"closed": False}
    destination = RNS.Destination(identity, RNS.Destination.IN, RNS.Destination.SINGLE, APP_NAME, ASPECT)

    def on_link_established(link):
        pkt = RNS.Packet(link, payload.encode("utf-8"), create_receipt=False)
        pkt.pack()
        RNS.Transport.cache(pkt, force_cache=True)
        log(f"EVENT cached_ready hash={RNS.hexrep(pkt.packet_hash, delimit=False)} payload={payload}")

        def on_closed(_link):
            done["closed"] = True

        link.set_link_closed_callback(on_closed)

    destination.set_link_established_callback(on_link_established)
    log("LISTEN_HASH " + RNS.hexrep(destination.hash, delimit=False))
    time.sleep(1.0)
    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if done["closed"]:
            return 0
        destination.announce()
        time.sleep(2.0)
    print("timeout waiting for cache flow", file=sys.stderr)
    return 1


def run_client(identity, destination_hex, hash_log_path, payload, wait_seconds):
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

    first_hash = await_hash_from_log(hash_log_path, wait_seconds)
    if first_hash is None:
        print("cached hash not found in log", file=sys.stderr)
        return 1
    time.sleep(0.5)
    RNS.Transport.cache_request(first_hash, link)
    log(f"EVENT cache_request hash={first_hash.hex()}")
    time.sleep(1.5)
    log(f"EVENT cache_request_sent hash={first_hash.hex()}")
    link.teardown()
    return 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default=None)
    parser.add_argument("--identity", default=None)
    parser.add_argument("--destination", default=None)
    parser.add_argument("--hash-log", default=None)
    parser.add_argument("--listen", action="store_true")
    parser.add_argument("--payload", default="cache-payload")
    parser.add_argument("--wait-seconds", type=float, default=30)
    args = parser.parse_args()

    reticulum = RNS.Reticulum(configdir=args.config, loglevel=RNS.LOG_DEBUG)
    _ = reticulum
    identity = prepare_identity(args.identity)

    if args.listen:
        raise SystemExit(run_listener(identity, args.payload, args.wait_seconds))

    if args.destination is None:
        print("destination is required", file=sys.stderr)
        raise SystemExit(1)

    raise SystemExit(run_client(identity, args.destination, args.hash_log, args.payload, args.wait_seconds))


def await_hash_from_log(path, timeout):
    if path is None:
        return None
    deadline = time.time() + timeout
    pattern = re.compile(r"^EVENT cached_ready hash=([0-9a-fA-F]+)\b")
    while time.time() < deadline:
        try:
            with open(path, "r", encoding="utf-8", errors="ignore") as fh:
                for line in fh:
                    m = pattern.match(line.strip())
                    if m:
                        return bytes.fromhex(m.group(1))
        except FileNotFoundError:
            pass
        time.sleep(0.1)
    return None


if __name__ == "__main__":
    main()
