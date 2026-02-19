#!/bin/bash

# Simple API benchmark for Cortex Memory.
# Benchmarks health, store, query, and delete endpoints.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Source common functions
source "${SCRIPT_DIR}/lib/common.sh"

API_URL="${CORTEX_API_URL:-http://localhost:9123}"
APP_ID="${CORTEX_APP_ID:-bench-app}"
USER_ID="${CORTEX_USER_ID:-bench-user}"
N="${1:-20}"

if ! is_positive_integer "$N"; then
    die "Anzahl muss eine positive Ganzzahl sein. Verwendung: $0 [anzahl]"
fi

if ! has_jq; then
    die "jq wird für benchmark.sh benötigt."
fi

summarize() {
    local name="$1"
    local data="$2"
    echo "$data" | tr ' ' '\n' | awk -v metric="$name" '
        NF {
            c++;
            sum += $1;
            if (min == "" || $1 < min) min = $1;
            if (max == "" || $1 > max) max = $1;
        }
        END {
            if (c > 0) {
                printf "%s n=%d avg=%.4fs min=%.4fs max=%.4fs\n", metric, c, sum/c, min, max;
            }
        }
    '
}

HEALTH_TIMES=""
STORE_TIMES=""
QUERY_TIMES=""
DELETE_TIMES=""

log_info "Starte Benchmark (N=${N}, API=${API_URL})..."

for i in $(seq 1 "$N"); do
    HEALTH_TIMES="$HEALTH_TIMES $(curl -s -o /dev/null -w '%{time_total}' "${API_URL}/health")"

    content="benchmark-memory-${i}-$(date +%s%N)"
    store_response=$(
        curl -s -w "\n%{time_total}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d "{\"appId\":\"${APP_ID}\",\"externalUserId\":\"${USER_ID}\",\"content\":\"${content}\",\"metadata\":{\"type\":\"benchmark\"}}" \
            "${API_URL}/seeds"
    )
    store_body=$(printf '%s' "$store_response" | sed '$d')
    store_time=$(printf '%s' "$store_response" | awk 'END { print $0 }')
    STORE_TIMES="$STORE_TIMES $store_time"

    id=$(extract_id "$store_body")
    if [ -z "$id" ] || [ "$id" = "null" ]; then
        die "Konnte keine ID aus Store-Response lesen."
    fi

    QUERY_TIMES="$QUERY_TIMES $(curl -s -o /dev/null -w '%{time_total}' \
        -X POST \
        -H 'Content-Type: application/json' \
        -d "{\"appId\":\"${APP_ID}\",\"externalUserId\":\"${USER_ID}\",\"query\":\"benchmark-memory-${i}\",\"limit\":5}" \
        "${API_URL}/seeds/query")"

    DELETE_TIMES="$DELETE_TIMES $(curl -s -o /dev/null -w '%{time_total}' \
        -X DELETE \
        -G \
        --data-urlencode "appId=${APP_ID}" \
        --data-urlencode "externalUserId=${USER_ID}" \
        "${API_URL}/seeds/${id}")"
done

echo ""
echo "Benchmark Ergebnisse (N=${N}, API=${API_URL})"
echo "=========================================="
summarize "health" "$HEALTH_TIMES"
summarize "store"  "$STORE_TIMES"
summarize "query"  "$QUERY_TIMES"
summarize "delete" "$DELETE_TIMES"
