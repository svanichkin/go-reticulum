#!/usr/bin/env python3
import argparse
import sys
import time

import RNS


class Observer:
    def __init__(self, aspect):
        self.aspect_filter = aspect
        self.count = 0
        self.first_at = None

    def received_announce(self, destination_hash, announced_identity, app_data):
        self.count += 1
        if self.first_at is None:
            self.first_at = time.time()
        print(f"OBSERVED count={self.count} destination={RNS.prettyhexrep(destination_hash)}", flush=True)


def main():
    parser = argparse.ArgumentParser(description="Observe announce callbacks on a shared instance")
    parser.add_argument("--config", required=True, help="Reticulum config dir")
    parser.add_argument("--aspect", required=True, help="announce aspect filter")
    parser.add_argument("--timeout", type=float, default=12.0, help="overall observer timeout")
    parser.add_argument("--settle", type=float, default=3.0, help="extra wait after first observe")
    parser.add_argument("--loglevel", type=int, default=1, help="Reticulum log level")
    args = parser.parse_args()

    RNS.Reticulum(configdir=args.config, loglevel=args.loglevel)
    observer = Observer(args.aspect)
    RNS.Transport.register_announce_handler(observer)

    print("READY", flush=True)

    deadline = time.time() + args.timeout
    while True:
        now = time.time()
        if observer.count > 0 and observer.first_at is not None and now >= observer.first_at + args.settle:
            break
        if now >= deadline:
            break
        time.sleep(0.1)

    print(f"COUNT={observer.count}", flush=True)


if __name__ == "__main__":
    main()
