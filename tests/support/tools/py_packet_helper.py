#!/usr/bin/env python3

import argparse
import os
import sys
import time

import RNS

APP_NAME = "paritypacket"


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


def run_listener(identity, mode, group_key_hex, expect_payload, wait_seconds, announce):
    if mode == "plain":
        destination = RNS.Destination(None, RNS.Destination.IN, RNS.Destination.PLAIN, APP_NAME, mode)
    elif mode == "group":
        destination = RNS.Destination(identity, RNS.Destination.IN, RNS.Destination.GROUP, APP_NAME, mode)
        destination.load_private_key(bytes.fromhex(group_key_hex))
    else:
        print("unsupported mode", file=sys.stderr)
        return 1

    received = {"done": False, "ok": False}

    def on_packet(data, _packet):
        payload = data.decode("utf-8", errors="replace")
        log("EVENT received " + payload)
        received["done"] = True
        received["ok"] = payload == expect_payload

    destination.set_packet_callback(on_packet)
    log("LISTEN_HASH " + RNS.hexrep(destination.hash, delimit=False))

    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        if announce and mode == "group":
            destination.announce()
        if received["done"]:
            return 0 if received["ok"] else 1
        time.sleep(1.0)

    print("timeout", file=sys.stderr)
    return 1


def run_sender(mode, destination_hex, group_key_hex, payload, wait_seconds, remote_identity_path):
    destination_hash = bytes.fromhex(destination_hex)

    if mode == "plain":
        destination = RNS.Destination(None, RNS.Destination.OUT, RNS.Destination.PLAIN, APP_NAME, mode)
    elif mode == "group":
        remote_identity = None
        if remote_identity_path:
            remote_identity = RNS.Identity.from_file(remote_identity_path)
        if remote_identity is None:
            if not await_path(destination_hash, wait_seconds):
                print("path not found", file=sys.stderr)
                return 1
            remote_identity = RNS.Identity.recall(destination_hash)
        if remote_identity is None:
            print("could not recall remote identity", file=sys.stderr)
            return 1
        destination = RNS.Destination(remote_identity, RNS.Destination.OUT, RNS.Destination.GROUP, APP_NAME, mode)
        destination.load_private_key(bytes.fromhex(group_key_hex))
    else:
        print("unsupported mode", file=sys.stderr)
        return 1

    packet = RNS.Packet(destination, payload.encode("utf-8"))
    log("EVENT destination " + RNS.hexrep(destination.hash, delimit=False))
    packet.send()
    log("EVENT sent")
    return 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default=None)
    parser.add_argument("--identity", default=None)
    parser.add_argument("--mode", required=True)
    parser.add_argument("--listen", action="store_true")
    parser.add_argument("--destination", default=None)
    parser.add_argument("--group-key", default="")
    parser.add_argument("--payload", default="hello")
    parser.add_argument("--wait-seconds", type=float, default=30)
    parser.add_argument("--announce", action="store_true")
    parser.add_argument("--remote-identity", default=None)
    args = parser.parse_args()

    _ = RNS.Reticulum(configdir=args.config, loglevel=2)
    identity = prepare_identity(args.identity)

    if args.listen:
        raise SystemExit(run_listener(identity, args.mode, args.group_key, args.payload, args.wait_seconds, args.announce))

    if args.destination is None:
        print("destination is required", file=sys.stderr)
        raise SystemExit(1)

    raise SystemExit(run_sender(args.mode, args.destination, args.group_key, args.payload, args.wait_seconds, args.remote_identity))


if __name__ == "__main__":
    main()
