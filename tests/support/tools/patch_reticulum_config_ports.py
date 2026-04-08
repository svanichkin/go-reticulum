#!/usr/bin/env python3
import argparse
import pathlib
import re


def find_reticulum_section(lines):
    start = None
    for i, line in enumerate(lines):
        if line.strip().lower() == "[reticulum]":
            start = i
            break
    if start is None:
        return None, None
    end = len(lines)
    for j in range(start + 1, len(lines)):
        s = lines[j].strip()
        if s.startswith("[") and s.endswith("]") and not s.startswith("[["):
            end = j
            break
    return start, end


def find_interfaces_section(lines):
    start = None
    for i, line in enumerate(lines):
        if line.strip().lower() == "[interfaces]":
            start = i
            break
    if start is None:
        return None, None
    end = len(lines)
    for j in range(start + 1, len(lines)):
        s = lines[j].strip()
        if s.startswith("[") and s.endswith("]") and not s.startswith("[["):
            end = j
            break
    return start, end


def set_key_in_section(lines, section_start, section_end, key, value):
    key_re = re.compile(rf"^(\s*){re.escape(key)}\s*=\s*.*$")
    for i in range(section_start + 1, section_end):
        m = key_re.match(lines[i])
        if m:
            indent = m.group(1)
            lines[i] = f"{indent}{key} = {value}\n"
            return
    # Insert before section end, keep a 2-space indent by default.
    insert_at = section_end
    indent = "  "
    lines.insert(insert_at, f"{indent}{key} = {value}\n")


def replace_key_in_range(lines, start, end, key, value):
    key_re = re.compile(rf"^(\s*){re.escape(key)}\s*=\s*.*$", re.IGNORECASE)
    for i in range(start + 1, end):
        m = key_re.match(lines[i])
        if not m:
            continue
        indent = m.group(1)
        lines[i] = f"{indent}{key} = {value}\n"


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--path", required=True, help="path to config file (in-place)")
    ap.add_argument("--shared-instance-type", default=None, help="override shared_instance_type (eg tcp/unix)")
    ap.add_argument("--rpc-key", default=None, help="override rpc_key (hex string)")
    ap.add_argument("--shared-instance-port", type=int, required=True)
    ap.add_argument("--instance-control-port", type=int, required=True)
    ap.add_argument("--interfaces-port-base", type=int, default=None)
    ap.add_argument("--listen-port", type=int, default=None, help="override listen_port in [interfaces]")
    ap.add_argument("--forward-port", type=int, default=None, help="override forward_port in [interfaces]")
    ap.add_argument("--target-port", type=int, default=None, help="override target_port in [interfaces]")
    args = ap.parse_args()

    path = pathlib.Path(args.path)
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines(keepends=True)

    sec_start, sec_end = find_reticulum_section(lines)
    if sec_start is None:
        # Create a reticulum section at the top if missing.
        lines = ["[reticulum]\n"] + (["\n"] if lines and lines[0].strip() else []) + lines
        sec_start, sec_end = find_reticulum_section(lines)

    set_key_in_section(lines, sec_start, sec_end, "shared_instance_port", str(args.shared_instance_port))
    set_key_in_section(lines, sec_start, sec_end, "instance_control_port", str(args.instance_control_port))
    if args.shared_instance_type is not None:
        set_key_in_section(lines, sec_start, sec_end, "shared_instance_type", str(args.shared_instance_type))
    if args.rpc_key is not None:
        set_key_in_section(lines, sec_start, sec_end, "rpc_key", str(args.rpc_key))

    if args.interfaces_port_base is not None:
        if_start, if_end = find_interfaces_section(lines)
        if if_start is not None:
            base = int(args.interfaces_port_base)
            replace_key_in_range(lines, if_start, if_end, "listen_port", str(base))
            replace_key_in_range(lines, if_start, if_end, "target_port", str(base))
            replace_key_in_range(lines, if_start, if_end, "forward_port", str(base + 1))

    if args.listen_port is not None or args.forward_port is not None or args.target_port is not None:
        if_start, if_end = find_interfaces_section(lines)
        if if_start is not None:
            if args.listen_port is not None:
                replace_key_in_range(lines, if_start, if_end, "listen_port", str(int(args.listen_port)))
            if args.forward_port is not None:
                replace_key_in_range(lines, if_start, if_end, "forward_port", str(int(args.forward_port)))
            if args.target_port is not None:
                replace_key_in_range(lines, if_start, if_end, "target_port", str(int(args.target_port)))

    path.write_text("".join(lines), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
