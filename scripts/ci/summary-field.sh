#!/usr/bin/env bash
#
# Print selected fields from JSON evidence files as `key=value` lines, for use in job summaries.
#
set -euo pipefail

if [[ ${#} -lt 1 ]]; then
  echo "usage: summary-field.sh <json-file>#<dotted.field.path> [...]" >&2
  exit 2
fi

printf '%s\0' "$@" | python3 -c '
import json
import pathlib
import sys

raw = sys.stdin.buffer.read()
specifications = [item.decode("utf-8") for item in raw.split(b"\0") if item]

for specification in specifications:
    if "#" not in specification:
        print(f"{specification}=-")
        continue

    file_part, _, dotted_path = specification.partition("#")
    leaf = dotted_path.rsplit(".", 1)[-1]

    document = None
    path = pathlib.Path(file_part)
    if path.is_file():
        try:
            document = json.loads(path.read_text())
        except (json.JSONDecodeError, OSError):
            document = None

    value = document
    for segment in dotted_path.split("."):
        if isinstance(value, dict) and segment in value:
            value = value[segment]
        else:
            value = None
            break

    if isinstance(value, (dict, list)) or value is None:
        print(f"{leaf}=-")
    else:
        print(f"{leaf}={value}")
'
