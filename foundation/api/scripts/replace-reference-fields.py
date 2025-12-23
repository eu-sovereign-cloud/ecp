#!/usr/bin/env python3
"""Replace Reference field types with ReferenceObject only within struct definitions."""

import pathlib
import re
import sys


def main() -> None:
    path = pathlib.Path(sys.argv[1])
    lines = path.read_text().splitlines()

    within_struct = False
    brace_depth = 0

    for index, line in enumerate(lines):
        stripped = line.strip()

        if not within_struct:
            if re.match(r"^type\s+\w+\s+struct\b", stripped):
                within_struct = True
                brace_depth = stripped.count("{") - stripped.count("}")
                if brace_depth <= 0:
                    within_struct = False
                continue
            continue

        brace_depth += stripped.count("{")
        brace_depth -= stripped.count("}")

        if not (stripped.startswith("//") or stripped == "" or stripped == "union" or stripped == "}"):
            lines[index] = re.sub(r"(?<!\w)Reference(?!\w)", "ReferenceObject", line)

        if brace_depth <= 0:
            within_struct = False

    path.write_text("\n".join(lines) + "\n")


if __name__ == "__main__":
    main()
