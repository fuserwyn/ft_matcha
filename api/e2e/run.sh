#!/usr/bin/env bash
set -euo pipefail

API_BASE="${E2E_API_BASE:-http://localhost:8080}"
echo "Running e2e against ${API_BASE}"

RUN_E2E=1 E2E_API_BASE="${API_BASE}" go test -v ./e2e
