#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

# Find duplicated Go in first-party sources (including tests); print to the terminal only (no ./report/).
# Calibrate against GoLand: Duplicated code fragment uses structural “units”; jscpd is
# line/token based—tune -l / -k against Settings | Editor | Inspections.
#
# Noise from short repeated httptest/assert fragments is reduced by a higher --min-tokens (-k),
# not by excluding *_test.go.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "${SCRIPT_DIR}/.."

# -p "**/*.go"              only Go sources
# JSCPD_IGNORE: jscpd keeps only the last -i if you repeat the flag—one comma-separated list.
# -r console               stdout table + clones only (default adds time reporter; avoid HTML dir)
# -m strict                 near-identical blocks (like GoLand without Editor | Duplicates anonymization)
# -l 10                     minimum duplicate length in lines (raise with -k if needed)
# -k 90                     minimum tokens (above ~70 drops small test scaffolding clones in this repo)
# Optional: --threshold N   only fail exit when duplication count ≥ N
# Optional: --exitCode 0    always exit 0 while still printing results

JSCPD_IGNORE='**/examples/**,.cache/**,**/.cache/**,**/node_modules/**,.venv/**,**/.venv/**,venv/**,**/venv/**,**/__pycache__/**,.tox/**,**/.tox/**,.mypy_cache/**,**/.mypy_cache/**,vendor/**,**/vendor/**'

exec jscpd . -p "**/*.go" \
  -i "${JSCPD_IGNORE}" \
  -r console \
  -m strict \
  -l 10 \
  -k 90
