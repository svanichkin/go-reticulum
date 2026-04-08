#!/usr/bin/env python3
import json
import sys


def _get(obj, *keys, default=None):
    for k in keys:
        if isinstance(obj, dict) and k in obj:
            return obj[k]
    return default


def _as_str(v):
    if v is None:
        return ""
    if isinstance(v, (int, float, bool)):
        return str(v)
    if isinstance(v, str):
        return v
    return str(v)


def _is_local_iface(name: str, typ: str) -> bool:
    if name.startswith("LocalInterface[") or name.startswith("Shared Instance["):
        return True
    if typ.startswith("Local"):
        return True
    return False


def _normalize_iface_name(name: str) -> str:
    if "[" not in name or "]" not in name:
        return name
    prefix, rest = name.split("[", 1)
    inner = rest.split("]", 1)[0]
    if "/" in inner:
        inner = inner.split("/", 1)[0]
    inner = inner.strip()
    if inner:
        return inner
    prefix = prefix.strip()
    return prefix or name


def main() -> int:
    raw = sys.stdin.read()
    start = raw.find("{")
    if start == -1:
        start = raw.find("[")
    if start > 0:
        raw = raw[start:]
    data = json.loads(raw)
    interfaces = _get(data, "interfaces", "Interfaces", default=[])
    if isinstance(interfaces, dict):
        # Be forgiving if something returns a map.
        interfaces = list(interfaces.values())
    if not isinstance(interfaces, list):
        interfaces = []

    rows = []
    for it in interfaces:
        if not isinstance(it, dict):
            continue
        name = _as_str(_get(it, "name", "Name"))
        typ = _as_str(_get(it, "type", "Type"))
        if _is_local_iface(name, typ):
            continue
        name = _normalize_iface_name(name)
        status = _as_str(_get(it, "status", "Status"))
        mode = _as_str(_get(it, "mode", "Mode"))
        rows.append((name, typ, status, mode))

    rows.sort()
    for name, typ, status, mode in rows:
        sys.stdout.write(f"{name}\t{typ}\t{status}\t{mode}\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
