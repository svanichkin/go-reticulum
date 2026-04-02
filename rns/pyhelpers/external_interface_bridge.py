import base64
import json
import os
import sys
import threading
import traceback


_stdout_lock = threading.Lock()


def emit(obj):
    with _stdout_lock:
        sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
        sys.stdout.flush()


def emit_error(message):
    emit({"type": "error", "message": str(message)})


class Owner:
    def inbound(self, data, _iface=None):
        if isinstance(data, bytearray):
            data = bytes(data)
        if not isinstance(data, (bytes, bytearray)):
            raise TypeError("external interface owner.inbound() expects bytes")
        emit({"type": "inbound", "data": base64.b64encode(data).decode("ascii")})


class Interface:
    MODE_FULL = 0x01
    MODE_POINT_TO_POINT = 0x02
    MODE_ACCESS_POINT = 0x03
    MODE_ROAMING = 0x04
    MODE_BOUNDARY = 0x05
    MODE_GATEWAY = 0x06

    DISCOVER_PATHS_FOR = [MODE_ACCESS_POINT, MODE_GATEWAY, MODE_ROAMING]

    DEFAULT_IFAC_SIZE = 8

    def __init__(self):
        self.name = ""
        self.online = False
        self.detached = False
        self.bitrate = 0
        self.HW_MTU = 0
        self.HWMTU = 0
        self.rxb = 0
        self.txb = 0
        self.mode = self.MODE_FULL
        self.OUT = True
        self.IN = False
        self.FWD = False
        self.RPT = False

    @staticmethod
    def get_config_obj(configuration):
        return configuration

    def optimise_mtu(self):
        return None

    def __str__(self):
        if getattr(self, "name", ""):
            return self.name
        return self.__class__.__name__


class _RNSShim:
    LOG_NONE = -1
    LOG_CRITICAL = 0
    LOG_ERROR = 1
    LOG_WARNING = 2
    LOG_NOTICE = 3
    LOG_INFO = 4
    LOG_VERBOSE = 5
    LOG_DEBUG = 6
    LOG_EXTREME = 7

    Transport = Owner()

    @staticmethod
    def log(message, _level=None):
        print(str(message), file=sys.stderr, flush=True)

    @staticmethod
    def trace_exception(exc):
        traceback.print_exception(type(exc), exc, exc.__traceback__, file=sys.stderr)

    @staticmethod
    def panic():
        raise SystemExit(255)


def main():
    if len(sys.argv) != 3:
        raise RuntimeError("usage: external_interface_bridge.py <module_path> <config_json>")

    module_path = sys.argv[1]
    configuration = json.loads(sys.argv[2])

    module_dir = os.path.dirname(os.path.abspath(module_path))
    if module_dir and module_dir not in sys.path:
        sys.path.insert(0, module_dir)

    interface_globals = {
        "__file__": module_path,
        "__name__": "__external_interface__",
        "Interface": Interface,
        "RNS": _RNSShim,
    }

    with open(module_path, "r", encoding="utf-8") as class_file:
        interface_code = class_file.read()

    exec(compile(interface_code, module_path, "exec"), interface_globals)
    interface_class = interface_globals.get("interface_class")
    if interface_class is None:
        raise RuntimeError(f"external interface {os.path.basename(module_path)} does not define interface_class")

    interface = interface_class(_RNSShim.Transport, configuration)

    if getattr(interface, "HWMTU", 0) == 0 and getattr(interface, "HW_MTU", 0):
        interface.HWMTU = int(interface.HW_MTU)

    ready = {
        "type": "ready",
        "name": getattr(interface, "name", ""),
        "interface_type": interface.__class__.__name__,
        "online": bool(getattr(interface, "online", False)),
        "bitrate": int(getattr(interface, "bitrate", 0) or 0),
        "hw_mtu": int(getattr(interface, "HWMTU", 0) or 0),
        "ifac_size": int(getattr(interface, "ifac_size", getattr(interface.__class__, "DEFAULT_IFAC_SIZE", 0)) or 0),
    }
    emit(ready)

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        command = json.loads(line)
        command_type = command.get("type")
        if command_type == "outgoing":
            try:
                payload = base64.b64decode(command.get("data", ""))
                interface.process_outgoing(payload)
            except Exception as exc:
                emit_error(str(exc))
        elif command_type == "detach":
            if hasattr(interface, "detach"):
                try:
                    interface.detach()
                except Exception as exc:
                    emit_error(str(exc))
            break


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        emit_error(str(exc))
        traceback.print_exception(type(exc), exc, exc.__traceback__, file=sys.stderr)
        raise
