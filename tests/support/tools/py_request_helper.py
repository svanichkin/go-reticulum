#!/usr/bin/env python3

import argparse
import os
import sys
import time

import RNS

APP_NAME = "parityreq"
ASPECT = "rpc"


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


def run_listener(identity, wait_seconds):
    destination = RNS.Destination(identity, RNS.Destination.IN, RNS.Destination.SINGLE, APP_NAME, ASPECT)

    def echo(path, data, request_id, link_id, remote_identity, requested_at):
        return f"echo:{data}"

    def sleep_handler(path, data, request_id, link_id, remote_identity, requested_at):
        secs = float(data)
        time.sleep(secs)
        return f"slept:{secs}"

    destination.register_request_handler(path="echo", response_generator=echo, allow=RNS.Destination.ALLOW_ALL)
    destination.register_request_handler(path="sleep", response_generator=sleep_handler, allow=RNS.Destination.ALLOW_ALL)

    log("LISTEN_HASH " + RNS.hexrep(destination.hash, delimit=False))
    time.sleep(1.0)
    deadline = time.time() + wait_seconds
    while time.time() < deadline:
        destination.announce()
        time.sleep(3.0)
    log("EVENT timeout")
    return 0


def run_client(identity, destination_hex, wait_seconds):
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
    time.sleep(0.2)

    echo_state = {"resp": None, "failed": False}

    def echo_ok(receipt):
        echo_state["resp"] = receipt.response

    def echo_failed(receipt):
        echo_state["failed"] = True

    rr = link.request("echo", "hello", response_callback=echo_ok, failed_callback=echo_failed, timeout=5)
    if rr is False:
        print("echo request not sent", file=sys.stderr)
        return 1

    deadline = time.time() + 8
    while time.time() < deadline:
        if echo_state["resp"] is not None:
            log(f"EVENT echo_response {echo_state['resp']}")
            break
        if echo_state["failed"]:
            print("echo request failed", file=sys.stderr)
            return 1
        time.sleep(0.1)
    else:
        print("echo response timeout", file=sys.stderr)
        return 1

    sleep_state = {"resp": None, "failed": False}

    def sleep_ok(receipt):
        sleep_state["resp"] = receipt.response

    def sleep_failed(receipt):
        sleep_state["failed"] = True

    rr = link.request("sleep", 3, response_callback=sleep_ok, failed_callback=sleep_failed, timeout=1)
    if rr is False:
        print("sleep request not sent", file=sys.stderr)
        return 1

    deadline = time.time() + 4
    while time.time() < deadline:
        if sleep_state["failed"]:
            print("sleep request unexpectedly failed", file=sys.stderr)
            return 1
        if sleep_state["resp"] is not None:
            log(f"EVENT sleep_response {sleep_state['resp']}")
            link.teardown()
            return 0
        time.sleep(0.1)

    print("sleep response timeout", file=sys.stderr)
    return 1


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default=None)
    parser.add_argument("--identity", default=None)
    parser.add_argument("--destination", default=None)
    parser.add_argument("--listen", action="store_true")
    parser.add_argument("--wait-seconds", type=float, default=30)
    args = parser.parse_args()

    reticulum = RNS.Reticulum(configdir=args.config, loglevel=2)
    _ = reticulum
    identity = prepare_identity(args.identity)

    if args.listen:
        raise SystemExit(run_listener(identity, args.wait_seconds))

    if args.destination is None:
        print("destination is required", file=sys.stderr)
        raise SystemExit(1)

    raise SystemExit(run_client(identity, args.destination, args.wait_seconds))


if __name__ == "__main__":
    main()
