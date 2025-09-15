#!/usr/bin/env bash
pwd
set -euo pipefail
cd "$(dirname "$0")"
exec go run ./gui/cmd/main.go "$@"