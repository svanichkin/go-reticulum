#!/usr/bin/env python3

import argparse
import os
import sys
import time

import RNS
from RNS.Channel import MessageBase
from RNS.vendor import umsgpack

APP_NAME = "paritychannel"
ASPECT = "messages"


class ChannelMessage(MessageBase):
    MSGTYPE = 0xABCD

    def __init__(self):
        self.id = ""
        self.data = ""

    def pack(self):
        return umsgpack.packb([self.id, self.data])

    def unpack(self, raw):
        unpacked = umsgpack.unpackb(raw)
        self.id = unpacked[0]
        self.data = unpacked[1]


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
    received = {"ids": ""}
    destination = RNS.Destination(identity, RNS.Destination.IN, RNS.Destination.SINGLE, APP_NAME, ASPECT)

    def on_link_established(link):
        channel = link.get_channel()
        channel.register_message_type(ChannelMessage)

        def on_message(message):
            log(f"EVENT received {message.id} {message.data}")
            received["ids"] += message.id
            return True

        channel.add_message_handler(on_message)

    destination.set_link_established_callback(on_link_established)
    log("LISTEN_HASH " + RNS.hexrep(destination.hash, delimit=False))
    deadline = time.time() + wait_seconds
    time.sleep(1.0)
    while time.time() < deadline:
        if received["ids"] == "123":
            return 0
        destination.announce()
        time.sleep(2.0)

    print(f"timeout ids={received['ids']}", file=sys.stderr)
    return 1


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
    time.sleep(1.0)

    channel = link.get_channel()
    channel.register_message_type(ChannelMessage)
    for msg_id, data in [("1", "alpha"), ("2", "beta"), ("3", "gamma")]:
        deadline = time.time() + wait_seconds
        while time.time() < deadline and not channel.is_ready_to_send():
            time.sleep(0.05)
        if not channel.is_ready_to_send():
            print(f"channel not ready for message {msg_id}", file=sys.stderr)
            return 1
        msg = ChannelMessage()
        msg.id = msg_id
        msg.data = data
        channel.send(msg)
        log(f"EVENT sent {msg_id} {data}")

    time.sleep(2.0)
    link.teardown()
    return 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default=None)
    parser.add_argument("--identity", default=None)
    parser.add_argument("--destination", default=None)
    parser.add_argument("--listen", action="store_true")
    parser.add_argument("--wait-seconds", type=float, default=45)
    args = parser.parse_args()

    _ = RNS.Reticulum(configdir=args.config, loglevel=2)
    identity = prepare_identity(args.identity)

    if args.listen:
        raise SystemExit(run_listener(identity, args.wait_seconds))

    if args.destination is None:
        print("destination is required", file=sys.stderr)
        raise SystemExit(1)

    raise SystemExit(run_client(identity, args.destination, args.wait_seconds))


if __name__ == "__main__":
    main()
