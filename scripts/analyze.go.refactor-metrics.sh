#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

# Refactoring backlog metrics for first-party Go, via golangci-lint.
# Uses cognitive + cyclomatic complexity, nesting depth, function length, and
# maintainability index — see scripts/golangci.refactor-metrics.yml.
#
# Requires golangci-lint v2+ (https://golangci-lint.run/).
#
# Focus one or more linters (same settings as in the YAML), e.g. duplicate strings:
#   ./scripts/analyze.go.refactor-metrics.sh --only goconst
#   REFACTOR_METRICS_ONLY=goconst ./scripts/analyze.go.refactor-metrics.sh
#   ./scripts/analyze.go.refactor-metrics.sh --only goconst,misspell
#
# Environment:
#   REFACTOR_METRICS_ONLY       comma-separated linters; CLI --only overrides this
#                                 run --list-linters to print enabled linters, or see scripts/golangci.refactor-metrics.yml
#                                 full list: https://golangci-lint.run/usage/linters/
#   REFACTOR_METRICS_NO_FAIL=1   always exit 0 (useful while triaging a large report)
#   REFACTOR_METRICS_FORMAT=json  machine-readable report on stdout (JSON object with Issues, etc.)
#   REFACTOR_METRICS_JSON=file     write JSON issues to file (still prints text to stdout unless FORMAT=json)
#   REFACTOR_METRICS_BOOTSTRAP=1  try to auto-install golangci-lint if missing (default: 1)
#
# Pass a directory that contains go.mod (e.g. test/jquery_example) to lint that
# module from its root; otherwise the repo root module and ./... are used.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/.." &>/dev/null && pwd)
CONFIG="${SCRIPT_DIR}/golangci.refactor-metrics.yml"

usage() {
  cat <<'EOF'
Refactoring / maintainability lint for first-party Go via golangci-lint
(see scripts/golangci.refactor-metrics.yml).

Usage:
  ./scripts/analyze.go.refactor-metrics.sh [options] [golangci-lint run flags...]

Options:
  --only, -o NAMES   run only these linters (comma-separated); e.g. goconst
  --list-linters     print linters enabled in golangci.refactor-metrics.yml and exit
  -h, --help         show this help

Environment:
  REFACTOR_METRICS_ONLY          same as --only if no --only on the command line
  REFACTOR_METRICS_NO_FAIL=1     always exit 0 while triaging
  REFACTOR_METRICS_FORMAT=json   JSON report on stdout
  REFACTOR_METRICS_JSON=path     write JSON issues to path
  REFACTOR_METRICS_BOOTSTRAP=1   try to auto-install golangci-lint if missing (default: 1)

Examples:
  ./scripts/analyze.go.refactor-metrics.sh --only goconst
  ./scripts/analyze.go.refactor-metrics.sh -o errcheck,errorlint
  ./scripts/analyze.go.refactor-metrics.sh test/jquery_example
EOF
}

list_linters_from_config() {
  awk '
    /^  enable:/ { in_enable = 1; next }
    in_enable && /^    - / { sub(/^    - /, ""); print }
    in_enable && /^  (settings|exclusions):/ { exit }
  ' "${CONFIG}"
}

if [[ "${1:-}" == "--list-linters" ]]; then
  list_linters_from_config
  exit 0
fi

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

cd "${REPO_ROOT}"

only_linters="${REFACTOR_METRICS_ONLY:-}"
passthrough=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --only=*)
      only_linters="${1#*=}"
      shift
      ;;
    --only | -o)
      if [[ -z "${2:-}" ]]; then
        echo "${0##*/}: --only requires a linter name (comma-separated for several)" >&2
        exit 2
      fi
      only_linters="$2"
      shift 2
      ;;
    *)
      passthrough+=("$1")
      shift
      ;;
  esac
done

# Passthrough: golangci-lint flags plus optional package/module paths. Directories that
# contain go.mod are separate modules; lint must run from that root so typecheck uses the
# correct module, not the repo root.
flags_only=()
module_roots=()
go_packages=()
# Bash 3.2 + set -u: "${passthrough[@]}" is unbound when the array is empty.
if [[ ${#passthrough[@]} -gt 0 ]]; then
  for a in "${passthrough[@]}"; do
    if [[ "${a}" == -* ]]; then
      flags_only+=("${a}")
      continue
    fi
    resolved="${a}"
    if [[ "${resolved}" != /* ]]; then
      resolved="${REPO_ROOT}/${resolved}"
    fi
    resolved="${resolved%/}"
    if [[ -f "${resolved}/go.mod" ]]; then
      module_roots+=("$(cd "${resolved}" && pwd)")
    else
      go_packages+=("${a}")
    fi
  done
fi

if [[ ${#module_roots[@]} -gt 0 && ${#go_packages[@]} -gt 0 ]]; then
  echo "${0##*/}: cannot mix separate-module directories (${module_roots[*]}) with repo-relative package paths (${go_packages[*]}); run those as two invocations." >&2
  exit 2
fi

if [[ ${#module_roots[@]} -gt 1 ]]; then
  if [[ "${REFACTOR_METRICS_FORMAT:-}" == "json" || -n "${REFACTOR_METRICS_JSON:-}" ]]; then
    echo "${0##*/}: JSON output is only supported for a single module directory at a time." >&2
    exit 2
  fi
fi

golangci_bin=""
refresh_golangci_bin() {
  golangci_bin=""
  if command -v golangci-lint >/dev/null 2>&1; then
    golangci_bin=$(command -v golangci-lint)
    return 0
  fi

  local gobin=""
  local gopath=""
  if command -v go >/dev/null 2>&1; then
    gobin="$(go env GOBIN 2>/dev/null || true)"
    gopath="$(go env GOPATH 2>/dev/null || true)"
  fi

  if [[ -n "${gobin}" && -x "${gobin}/golangci-lint" ]]; then
    golangci_bin="${gobin}/golangci-lint"
    return 0
  fi
  if [[ -n "${gopath}" && -x "${gopath}/bin/golangci-lint" ]]; then
    golangci_bin="${gopath}/bin/golangci-lint"
    return 0
  fi
  return 1
}

try_install_golangci() {
  [[ "${REFACTOR_METRICS_BOOTSTRAP:-1}" == "1" ]] || return 1

  echo "golangci-lint not found. Trying to install automatically..." >&2

  if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
    echo "Attempting install via Homebrew..." >&2
    if brew install golangci-lint >/dev/null 2>&1; then
      refresh_golangci_bin && return 0
    fi
    echo "Homebrew install did not succeed; trying Go install fallback..." >&2
  fi

  if command -v go >/dev/null 2>&1; then
    echo "Attempting install via go install..." >&2
    if go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest >/dev/null 2>&1; then
      local gobin
      local gopath
      gobin="$(go env GOBIN 2>/dev/null || true)"
      gopath="$(go env GOPATH 2>/dev/null || true)"
      if [[ -n "${gobin}" ]]; then
        export PATH="${gobin}:${PATH}"
      elif [[ -n "${gopath}" ]]; then
        export PATH="${gopath}/bin:${PATH}"
      fi
      refresh_golangci_bin && return 0
    fi
  fi

  return 1
}

refresh_golangci_bin || true
if [[ -z "${golangci_bin}" ]]; then
  try_install_golangci || true
  refresh_golangci_bin || true
fi

if [[ -z "${golangci_bin}" ]]; then
  echo "golangci-lint not found after auto-install attempt." >&2
  if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
    echo "Try: brew install golangci-lint" >&2
  fi
  if command -v go >/dev/null 2>&1; then
    echo "Try: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" >&2
  fi
  echo "Set REFACTOR_METRICS_BOOTSTRAP=0 to disable auto-install attempts." >&2
  exit 127
fi

# Keep a non-empty argv array before "${arr[@]}" so set -u is safe on Bash 3.2
# (empty array expansion is "unbound" there).
# Treat lint findings as non-blocking unless BUILD_STOP_ON_LINT_FINDINGS=1.
if [[ "${BUILD_STOP_ON_LINT_FINDINGS:-0}" != "1" ]]; then
  REFACTOR_METRICS_NO_FAIL=1
fi

golangci_cmd=(run -c "${CONFIG}")
if [[ -n "${only_linters}" ]]; then
  golangci_cmd+=(--enable-only "${only_linters}")
fi
if [[ "${REFACTOR_METRICS_NO_FAIL:-}" == "1" ]]; then
  golangci_cmd+=(--issues-exit-code=0)
fi

# Bash 3.2 + set -u: "${empty[@]}" is unbound — merge into one array before expanding.
full_golangci_cmd=("${golangci_cmd[@]}")
[[ ${#flags_only[@]} -gt 0 ]] && full_golangci_cmd+=("${flags_only[@]}")

run_golangci() {
  local workdir="$1"
  shift
  (cd "${workdir}" && exec "${golangci_bin}" "$@")
}

if [[ ${#module_roots[@]} -gt 0 ]]; then
  status=0
  for mod_root in "${module_roots[@]}"; do
    if [[ "${REFACTOR_METRICS_FORMAT:-}" == "json" ]]; then
      run_golangci "${mod_root}" "${full_golangci_cmd[@]}" \
        --output.json.path=stdout \
        --output.text.path=stderr \
        --show-stats=false \
        ./... || status=$?
    elif [[ -n "${REFACTOR_METRICS_JSON:-}" ]]; then
      mkdir -p "$(dirname "${REFACTOR_METRICS_JSON}")"
      run_golangci "${mod_root}" "${full_golangci_cmd[@]}" \
        --output.json.path="${REFACTOR_METRICS_JSON}" \
        ./... || status=$?
    else
      run_golangci "${mod_root}" "${full_golangci_cmd[@]}" ./... || status=$?
    fi
  done
  exit "${status}"
fi

if [[ "${REFACTOR_METRICS_FORMAT:-}" == "json" ]]; then
  if [[ ${#go_packages[@]} -gt 0 ]]; then
    run_golangci "${REPO_ROOT}" "${full_golangci_cmd[@]}" \
      --output.json.path=stdout \
      --output.text.path=stderr \
      --show-stats=false \
      ./... "${go_packages[@]}"
  else
    run_golangci "${REPO_ROOT}" "${full_golangci_cmd[@]}" \
      --output.json.path=stdout \
      --output.text.path=stderr \
      --show-stats=false \
      ./...
  fi
  exit $?
fi

if [[ -n "${REFACTOR_METRICS_JSON:-}" ]]; then
  mkdir -p "$(dirname "${REFACTOR_METRICS_JSON}")"
  if [[ ${#go_packages[@]} -gt 0 ]]; then
    run_golangci "${REPO_ROOT}" "${full_golangci_cmd[@]}" \
      --output.json.path="${REFACTOR_METRICS_JSON}" \
      ./... "${go_packages[@]}"
  else
    run_golangci "${REPO_ROOT}" "${full_golangci_cmd[@]}" \
      --output.json.path="${REFACTOR_METRICS_JSON}" \
      ./...
  fi
  exit $?
fi

if [[ ${#go_packages[@]} -gt 0 ]]; then
  run_golangci "${REPO_ROOT}" "${full_golangci_cmd[@]}" ./... "${go_packages[@]}"
else
  run_golangci "${REPO_ROOT}" "${full_golangci_cmd[@]}" ./...
fi
exit $?
