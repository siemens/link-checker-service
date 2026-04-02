#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

# PMD Copy/Paste Detector on first-party Go; report on stdout only (no --report-file).
#
# Scans: cmd/, server/, infrastructure/, and repo-root *.go (non-recursive).
# Skips test/ sample programs and other trees outside those roots.
#
# Requires PMD CLI (https://pmd.github.io/): brew install pmd
#
# Environment:
#   PMD_CPD_NO_FAIL=1   print report but exit 0 when duplicates are found
#
# Optional: --no-fail-on-error       exit 0 on recoverable collection/token errors (pass through "$@")

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/.." &>/dev/null && pwd)
cd "${REPO_ROOT}"

if ! command -v pmd >/dev/null 2>&1; then
  echo "pmd not found. Install e.g.:" >&2
  echo "  brew install pmd" >&2
  exit 127
fi

pmd_args=(
  cpd --minimum-tokens 70 --language go
  -d cmd,server,infrastructure
  -d . --non-recursive
  --format text
)
if [[ "${PMD_CPD_NO_FAIL:-}" == "1" ]]; then
  pmd_args+=(--no-fail-on-violation)
fi

# --minimum-tokens 70   fragment size floor (try ~55–90 when calibrating vs IDE/jscpd)
# --language go         tokenizer for Go
# -d cmd,server,...     recurse into each first-party package tree
# -d . --non-recursive  repo-root *.go only (main.go, *_test.go)
# --format text         human-readable on stdout

exec pmd "${pmd_args[@]}" "$@"
