#!/usr/bin/env bash
set -Eeuo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8081}"
BASE_URL="${BASE_URL%/}"

check_endpoint() {
    local path="$1"
    local expected="$2"
    local response
    local status

    if ! response="$(curl --silent --show-error \
        --max-time 10 \
        --write-out $'\n%{http_code}' \
        "${BASE_URL}${path}")"; then
        echo "FAIL: ${path} request failed" >&2
        return 1
    fi

    status="${response##*$'\n'}"
    response="${response%$'\n'*}"

    if [[ "$status" != "200" ]]; then
        echo "FAIL: ${path} returned HTTP ${status}" >&2
        return 1
    fi

    if ! grep -Fq "$expected" <<< "$response"; then
        echo "FAIL: ${path} returned an unexpected response" >&2
        return 1
    fi

    echo "PASS: ${path} returned HTTP ${status}"
}

check_endpoint "/healthz" '"status":"ok"'
check_endpoint "/readyz" '"status":"ready"'

echo "Deployment smoke tests passed."
