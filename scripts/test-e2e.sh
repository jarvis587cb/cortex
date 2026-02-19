#!/bin/bash

# End-to-End Test Script für Cortex Memory API
# Testet den gesamten Workflow gegen die laufende API

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Source common functions
source "${SCRIPT_DIR}/lib/common.sh"

# Konfiguration
API_URL="${CORTEX_API_URL:-http://localhost:9123}"
APP_ID="${CORTEX_APP_ID:-e2e-test-app}"
USER_ID="${CORTEX_USER_ID:-e2e-test-user}"

# Zähler
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Hilfsfunktionen
log_test() {
    echo -e "\n${YELLOW}=== $1 ===${NC}"
}

test_pass() {
    log_success "$1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

test_fail() {
    log_error "$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# Prüfe Dependencies
check_dependencies() {
    log_info "Prüfe Dependencies..."
    
    if ! command -v curl &> /dev/null; then
        test_fail "curl ist nicht installiert"
        exit 1
    fi
    
    if ! has_jq; then
        test_fail "jq ist nicht installiert"
        exit 1
    fi
    
    log_success "Alle Dependencies vorhanden"
}

# Test: Health Check
test_health() {
    log_test "Health Check"
    
    response=$(curl_with_status "${API_URL}/health")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        if echo "$body" | jq -e '.status == "ok"' >/dev/null 2>&1; then
            test_pass "Health Check erfolgreich"
        else
            test_fail "Health Check: Ungültige Response"
        fi
    else
        test_fail "Health Check: HTTP $http_code"
    fi
}

# Test: Memory speichern
test_store() {
    log_test "Memory speichern"
    
    test_content="e2e-test-$(date +%s)"
    json_body=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg content "$test_content" \
        '{appId: $appId, externalUserId: $externalUserId, content: $content, metadata: {}}')
    
    response=$(curl_with_status "${API_URL}/seeds" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$json_body")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        memory_id=$(extract_id "$body")
        if [ -n "$memory_id" ] && [ "$memory_id" != "null" ]; then
            test_pass "Memory gespeichert (ID: $memory_id)"
            echo "$memory_id" > /tmp/cortex_test_id.txt
        else
            test_fail "Memory speichern: Keine ID in Response"
        fi
    else
        test_fail "Memory speichern: HTTP $http_code"
    fi
}

# Test: Memory-Suche
test_query() {
    log_test "Memory-Suche"
    
    test_content="e2e-test-query-$(date +%s)"
    json_body=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg content "$test_content" \
        '{appId: $appId, externalUserId: $externalUserId, content: $content, metadata: {}}')
    
    # Erst speichern
    curl_with_status "${API_URL}/seeds" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$json_body" >/dev/null
    
    # Dann suchen
    query_body=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg query "e2e-test-query" \
        --argjson limit 5 \
        '{appId: $appId, externalUserId: $externalUserId, query: $query, limit: $limit}')
    
    response=$(curl_with_status "${API_URL}/seeds/query" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$query_body")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        count=$(count_items "$body")
        if [ "$count" -gt 0 ]; then
            test_pass "Memory-Suche erfolgreich (Gefunden: $count)"
        else
            test_fail "Memory-Suche: Keine Ergebnisse"
        fi
    else
        test_fail "Memory-Suche: HTTP $http_code"
    fi
}

# Test: Memory löschen
test_delete() {
    log_test "Memory löschen"
    
    if [ ! -f /tmp/cortex_test_id.txt ]; then
        test_fail "Memory löschen: Keine Test-ID gefunden"
        return
    fi
    
    memory_id=$(cat /tmp/cortex_test_id.txt)
    
    response=$(curl_with_status "${API_URL}/seeds/${memory_id}?appId=${APP_ID}&externalUserId=${USER_ID}" \
        -X DELETE)
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        deleted_id=$(extract_id "$body")
        if [ "$deleted_id" = "$memory_id" ]; then
            test_pass "Memory gelöscht (ID: $memory_id)"
        else
            test_fail "Memory löschen: Falsche ID in Response"
        fi
    elif [ "$http_code" = "404" ]; then
        test_fail "Memory löschen: Nicht gefunden (ID: $memory_id)"
    else
        test_fail "Memory löschen: HTTP $http_code"
    fi
}

# Test: Statistiken
test_stats() {
    log_test "Statistiken"
    
    response=$(curl_with_status "${API_URL}/stats")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        if echo "$body" | jq -e '.memories >= 0 and .entities >= 0 and .relations >= 0' >/dev/null 2>&1; then
            test_pass "Statistiken erfolgreich abgerufen"
        else
            test_fail "Statistiken: Ungültige Response-Struktur"
        fi
    else
        test_fail "Statistiken: HTTP $http_code"
    fi
}

# Cleanup
cleanup() {
    rm -f /tmp/cortex_test_id.txt
}

# Main
trap cleanup EXIT

log_info "Starte E2E-Tests gegen ${API_URL}"
log_info "App-ID: ${APP_ID}, User-ID: ${USER_ID}"

check_dependencies
test_health
test_store
test_query
test_delete
test_stats

# Zusammenfassung
echo ""
echo "=========================================="
echo "Test-Zusammenfassung"
echo "=========================================="
echo "Gesamt: $TOTAL_TESTS"
log_success "Bestanden: $TESTS_PASSED"
if [ $TESTS_FAILED -gt 0 ]; then
    log_error "Fehlgeschlagen: $TESTS_FAILED"
    exit 1
else
    log_success "Alle Tests bestanden!"
    exit 0
fi
