#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

echo "Updating Go Dependencies"

go get -u ./... && go get -u -t ./... && go test ./... && go mod tidy
