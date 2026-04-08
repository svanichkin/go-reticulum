#!/usr/bin/env python3
import argparse
import subprocess
import sys


def main() -> int:
    ap = argparse.ArgumentParser(add_help=False)
    ap.add_argument("--timeout", type=float, required=True)
    ap.add_argument("--", dest="double_dash", action="store_true")
    args, rest = ap.parse_known_args()

    cmd = rest
    if cmd and cmd[0] == "--":
        cmd = cmd[1:]
    if not cmd:
        sys.stderr.write("timeout_exec.py: missing command\n")
        return 2

    try:
        proc = subprocess.run(cmd, timeout=args.timeout)
        return proc.returncode
    except subprocess.TimeoutExpired:
        return 124


if __name__ == "__main__":
    raise SystemExit(main())

