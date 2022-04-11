#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

cd public

python3 -m http.server 8092
