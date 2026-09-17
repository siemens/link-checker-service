#!/usr/bin/env bash
# The single quality gate: formatting, static analysis (vet + golangci-lint),
# and the unit tests. Exits 0 only when everything is green. Mirrors the CI
# checks so the same gate can be run locally before pushing.
#
# golangci-lint is run via `go run` with a pinned version, so no separate
# install step is required.
set -euo pipefail
cd "$(dirname "$0")/.."

GOLANGCI=(go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2)

RED=$'\033[31m'; GREEN=$'\033[32m'; BOLD=$'\033[1m'; RESET=$'\033[0m'
section() { printf '\n%s\n' "${BOLD}==> $*${RESET}"; }
fail()    { printf '%serror:%s %s\n' "$RED" "$RESET" "$*" >&2; exit 1; }

section "gofmt"
unformatted="$(gofmt -l .)"
[[ -z "$unformatted" ]] || { printf '%s\n' "$unformatted" >&2; fail "gofmt: the files above need formatting (gofmt -w .)"; }

section "go vet"
go vet ./...

section "golangci-lint"
"${GOLANGCI[@]}" run --timeout=5m ./...

section "tests"
go test ./...

printf '\n%sQUALITY GATE PASSED%s\n' "$GREEN" "$RESET"
