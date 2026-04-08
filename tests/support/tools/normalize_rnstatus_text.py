#!/usr/bin/env python3
import re
import sys


def normalize(text: str) -> str:
    # Drop carriage returns used for spinner/progress updates.
    text = text.replace("\r", "")

    # Drop Reticulum log lines that may be printed when the shared instance is chatty.
    text = re.sub(r"^\[\d{4}-\d{2}-\d{2} [0-9:]{8}\] \[[A-Za-z]+\].*$\n?", "", text, flags=re.MULTILINE)

    # Normalise hashes printed as <deadbeef...>
    text = re.sub(r"<[0-9a-f]{8,}>", "<HASH>", text, flags=re.IGNORECASE)

    # Normalise interface names that embed ephemeral ports/ids.
    text = re.sub(r"LocalInterface\[\d+\]", "LocalInterface[PORT]", text)

    # Normalise traffic counters and rates (they are timing-dependent and may
    # differ slightly between Python and Go implementations).
    text = re.sub(r"^\s+Traffic\s+:\s+.*$", "    Traffic   : <TRAFFIC>", text, flags=re.MULTILINE)
    text = re.sub(r"^\s+↓.*$", "                <TRAFFIC>", text, flags=re.MULTILINE)
    text = re.sub(r"^\s+Totals\s+:\s+.*$", " Totals       : <TRAFFIC>", text, flags=re.MULTILINE)
    text = re.sub(r"^\s+Announces\s+:\s+.*$", "    Announces : <ANNOUNCES>", text, flags=re.MULTILINE)
    text = re.sub(r"^\s+<.*>↓.*$", "                <ANNOUNCES>", text, flags=re.MULTILINE)

    # Normalise uptime line (wall-clock dependent).
    text = re.sub(r"^ Uptime is .*$", " Uptime is <TIME>", text, flags=re.MULTILINE)

    # Normalise "expires <timestamp>" fragments if present.
    text = re.sub(r"\bexpires\s+\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\b", "expires <TIME>", text)

    # Trim trailing whitespace and collapse multiple blank lines.
    lines = [ln.rstrip() for ln in text.splitlines()]
    out = "\n".join(lines).strip() + "\n"
    out = re.sub(r"\n{3,}", "\n\n", out)
    return out


def main() -> int:
    data = sys.stdin.read()
    sys.stdout.write(normalize(data))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
