#!/usr/bin/env python3
import argparse
import os
import selectors
import sys


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--rx", required=True)
    parser.add_argument("--tx", required=True)
    args = parser.parse_args()

    rx_fd = os.open(args.rx, os.O_RDWR | os.O_NONBLOCK)
    tx_fd = os.open(args.tx, os.O_RDWR)

    sel = selectors.DefaultSelector()
    sel.register(0, selectors.EVENT_READ, "stdin")
    sel.register(rx_fd, selectors.EVENT_READ, "rx")

    stdout_fd = sys.stdout.fileno()
    stdin_open = True

    try:
        while True:
            events = sel.select(timeout=1.0)
            if not events and not stdin_open:
                return 0

            for key, _ in events:
                if key.data == "stdin":
                    data = os.read(0, 65536)
                    if not data:
                        stdin_open = False
                        sel.unregister(0)
                        continue
                    os.write(tx_fd, data)
                else:
                    try:
                        data = os.read(rx_fd, 65536)
                    except BlockingIOError:
                        continue
                    if not data:
                        continue
                    os.write(stdout_fd, data)
    finally:
        os.close(rx_fd)
        os.close(tx_fd)


if __name__ == "__main__":
    raise SystemExit(main())
