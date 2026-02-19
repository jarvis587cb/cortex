#!/bin/bash

# Cortex CLI Tool
# Bash-Script für die Cortex Memory API

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Source common functions
source "${SCRIPT_DIR}/lib/common.sh"

# Konfiguration
API_URL="${CORTEX_API_URL:-http://localhost:9123}"
APP_ID="${CORTEX_APP_ID:-openclaw}"
USER_ID="${CORTEX_USER_ID:-default}"

# Hilfsfunktion: JSON-String sicher erstellen
create_json() {
    local app_id="$1"
    local user_id="$2"
    local content="$3"
    local metadata="${4:-{}}"
    
    if ! has_jq; then
        error "jq ist erforderlich für cortex-cli.sh. Bitte installiere jq: sudo apt-get install jq"
    fi
    
    # Verwende jq für sicheres JSON-Erstellen
    if echo "$metadata" | jq . >/dev/null 2>&1; then
        jq -n \
            --arg appId "$app_id" \
            --arg externalUserId "$user_id" \
            --arg content "$content" \
            --argjson metadata "$metadata" \
            '{appId: $appId, externalUserId: $externalUserId, content: $content, metadata: $metadata}'
    else
        jq -n \
            --arg appId "$app_id" \
            --arg externalUserId "$user_id" \
            --arg content "$content" \
            '{appId: $appId, externalUserId: $externalUserId, content: $content, metadata: {}}'
    fi
}

# Alias für Kompatibilität
error() {
    die "$1"
}

success() {
    log_success "$1"
}

info() {
    log_info "$1"
}

# Health Check
health() {
    info "Prüfe API-Status..."
    response=$(curl_with_status "${API_URL}/health")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        echo "$body" | format_json
        success "API ist erreichbar"
    else
        error "API nicht erreichbar (HTTP $http_code)"
    fi
}

# Memory speichern (Seeds-API)
store() {
    local content="$1"
    local metadata="${2:-{}}"
    
    if [ -z "$content" ]; then
        error "Content darf nicht leer sein"
    fi
    
    info "Speichere Memory..."
    
    json_body=$(create_json "${APP_ID}" "${USER_ID}" "$content" "$metadata")
    
    response=$(curl_with_status "${API_URL}/seeds" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$json_body")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        echo "$body" | format_json
        memory_id=$(extract_id "$body")
        if [ -n "$memory_id" ]; then
            success "Memory gespeichert (ID: $memory_id)"
        fi
    else
        error "Fehler beim Speichern (HTTP $http_code): $body"
    fi
}

# Memory-Suche (Seeds-API)
query() {
    local query_text="$1"
    local limit="${2:-5}"
    
    if [ -z "$query_text" ]; then
        error "Query-Text darf nicht leer sein"
    fi
    
    if ! is_positive_integer "$limit"; then
        error "Limit muss eine positive Ganzzahl sein"
    fi
    
    info "Suche nach: $query_text (Limit: $limit)..."
    
    json_body=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg query "$query_text" \
        --argjson limit "$limit" \
        '{appId: $appId, externalUserId: $externalUserId, query: $query, limit: $limit}')
    
    response=$(curl_with_status "${API_URL}/seeds/query" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$json_body")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        echo "$body" | format_json
        count=$(count_items "$body")
        success "Gefunden: $count Memories"
    else
        error "Fehler bei der Suche (HTTP $http_code): $body"
    fi
}

# Memory löschen (Seeds-API)
delete() {
    local id="$1"
    
    if ! is_positive_integer "$id"; then
        error "ID muss eine positive Ganzzahl sein"
    fi
    
    info "Lösche Memory (ID: $id)..."
    
    response=$(curl_with_status "${API_URL}/seeds/${id}?appId=${APP_ID}&externalUserId=${USER_ID}" \
        -X DELETE)
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        echo "$body" | format_json
        success "Memory gelöscht"
    elif [ "$http_code" = "404" ]; then
        error "Memory nicht gefunden (ID: $id)"
    else
        error "Fehler beim Löschen (HTTP $http_code): $body"
    fi
}

# Statistiken abrufen
stats() {
    info "Lade Statistiken..."
    
    response=$(curl_with_status "${API_URL}/stats")
    parsed=$(parse_http_response "$response")
    body=$(echo "$parsed" | head -n -1)
    http_code=$(echo "$parsed" | tail -n 1)
    
    if [ "$http_code" = "200" ]; then
        echo "$body" | format_json
    else
        error "Fehler beim Laden der Statistiken (HTTP $http_code): $body"
    fi
}

# Hilfe anzeigen
help() {
    cat <<EOF
Cortex CLI Tool

Verwendung:
  $0 <command> [args...]

Befehle:
  health                    - Prüft API-Status
  store <content> [metadata] - Speichert ein Memory
  query <text> [limit]      - Sucht nach Memories (Standard-Limit: 5)
  delete <id>               - Löscht ein Memory
  stats                     - Zeigt Statistiken
  help                      - Zeigt diese Hilfe

Umgebungsvariablen:
  CORTEX_API_URL     - API Base URL (Standard: http://localhost:9123)
  CORTEX_APP_ID      - App-ID für Multi-Tenant (Standard: openclaw)
  CORTEX_USER_ID    - User-ID für Multi-Tenant (Standard: default)

Beispiele:
  $0 health
  $0 store "Der Nutzer mag Kaffee"
  $0 store "Präferenz" '{"tags":["preference"]}'
  $0 query "Kaffee" 10
  $0 delete 1
  $0 stats
EOF
}

# Main
case "${1:-help}" in
    health)
        health
        ;;
    store)
        if [ $# -lt 2 ]; then
            error "Verwendung: $0 store <content> [metadata]"
        fi
        store "${2}" "${3:-{}}"
        ;;
    query)
        if [ $# -lt 2 ]; then
            error "Verwendung: $0 query <text> [limit]"
        fi
        query "${2}" "${3:-5}"
        ;;
    delete)
        if [ $# -lt 2 ]; then
            error "Verwendung: $0 delete <id>"
        fi
        delete "${2}"
        ;;
    stats)
        stats
        ;;
    help|--help|-h)
        help
        ;;
    *)
        error "Unbekannter Befehl: $1\n\n$(help)"
        ;;
esac
