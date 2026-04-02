#!/usr/bin/env bash
# Requires bash (arrays, pipefail). Do not run with dash/sh.
if [ -z "${BASH_VERSION:-}" ]; then
  echo "This script must be run with bash (e.g. bash \"$0\" …)." >&2
  exit 1
fi
set -euo pipefail
IFS=$'\n\t'

# Refactoring backlog metrics via golangci-lint.
# Uses cognitive + cyclomatic complexity, nesting depth, function length, and
# maintainability index — see scripts/golangci.refactor-metrics.yml.
#
# Requires golangci-lint v2+ (https://golangci-lint.run/).
#
# Environment:
#   REFACTOR_METRICS_NO_FAIL=1   always exit 0 (useful while triaging a large report)
#   REFACTOR_METRICS_FORMAT=json  machine-readable report on stdout (JSON object with Issues, etc.)
#   REFACTOR_METRICS_JSON=file     write JSON issues to file (still prints text to stdout unless FORMAT=json)
#
# Positional args: optional package paths (same as `go test`). Default is ./... (this module only;
# nested go.mod trees under e.g. test/ are not included). If you pass paths, only those are analyzed
# — they replace the default, they are not combined with ./...

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/.." &>/dev/null && pwd)
CONFIG="${SCRIPT_DIR}/golangci.refactor-metrics.yml"

cd "${REPO_ROOT}"

golangci_bin=""
if command -v golangci-lint >/dev/null 2>&1; then
  golangci_bin=$(command -v golangci-lint)
else
  _gopath=""
  if command -v go >/dev/null 2>&1; then
    _gopath="$(go env GOPATH 2>/dev/null || true)"
  fi
  if [[ -n "${_gopath}" && -x "${_gopath}/bin/golangci-lint" ]]; then
    golangci_bin="${_gopath}/bin/golangci-lint"
  fi
fi

if [[ -z "${golangci_bin}" ]]; then
  echo "golangci-lint not found. Install v2+ e.g.:" >&2
  echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" >&2
  echo "  brew install golangci-lint   # macOS" >&2
  echo "  apt install golangci-lint    # Debian/Ubuntu (if packaged)" >&2
  exit 127
fi

extra_args=()
if [[ "${REFACTOR_METRICS_NO_FAIL:-}" == "1" ]]; then
  extra_args+=(--issues-exit-code=0)
fi

if [[ $# -gt 0 ]]; then
  scope_args=("$@")
else
  scope_args=(./...)
fi

# Bash 3.2 + set -u: "${extra_args[@]}" errors when the array is empty ("unbound variable").
if [[ "${REFACTOR_METRICS_FORMAT:-}" == "json" ]]; then
  exec "${golangci_bin}" run -c "${CONFIG}" "${extra_args[@]+"${extra_args[@]}"}" \
    --output.json.path=stdout \
    --output.text.path=stderr \
    --show-stats=false \
    "${scope_args[@]}"
fi

if [[ -n "${REFACTOR_METRICS_JSON:-}" ]]; then
  mkdir -p "$(dirname "${REFACTOR_METRICS_JSON}")"
  "${golangci_bin}" run -c "${CONFIG}" "${extra_args[@]+"${extra_args[@]}"}" \
    --output.json.path="${REFACTOR_METRICS_JSON}" \
    "${scope_args[@]}"
  exit $?
fi

exec "${golangci_bin}" run -c "${CONFIG}" "${extra_args[@]+"${extra_args[@]}"}" "${scope_args[@]}"
