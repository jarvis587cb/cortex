#!/bin/bash
# Cortex Memory Script - Neutron-compatible interface for OpenClaw
# Usage: ./scripts/cortex-memory.sh <command> [args...]
# Commands: test, save, search, context-create, context-list, context-get

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

# Config: Cortex URL and tenant (OpenClaw can use NEUTRON_AGENT_ID / YOUR_AGENT_IDENTIFIER or Cortex vars)
API_URL="${CORTEX_API_URL:-${NEUTRON_API_URL:-http://localhost:9123}}"
APP_ID="${CORTEX_APP_ID:-${NEUTRON_AGENT_ID:-openclaw}}"
USER_ID="${CORTEX_USER_ID:-${YOUR_AGENT_IDENTIFIER:-default}}"

# Alias for compatibility
info() { log_info "$1"; }
success() { log_success "$1"; }
error() { die "$1"; }

# test - Verify installation and credentials (like neutron-memory.sh test)
cmd_test() {
    log_info "Testing Cortex connection..."
    local response body http_code
    response=$(curl -s -w "\n%{http_code}" "${API_URL}/health" 2>/dev/null || echo -e "\n000")
    body=$(echo "$response" | head -n -1)
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" = "200" ]; then
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        log_success "Cortex API is reachable at $API_URL"
    else
        die "Cortex API not reachable (HTTP $http_code). Check CORTEX_API_URL and that the server is running."
    fi
}

# save - Save a memory (like neutron-memory.sh save "content" "metadata/tag")
cmd_save() {
    local content="${1:-}"
    local metadata="${2:-{}}"
    [ -z "$content" ] && die "Usage: $0 save \"<content>\" [metadata_json]"
    if ! has_jq 2>/dev/null; then
        metadata="{}"
    else
        echo "$metadata" | jq . >/dev/null 2>&1 || metadata="{}"
    fi
    local json
    json=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg content "$content" \
        --argjson metadata "$metadata" \
        '{appId: $appId, externalUserId: $externalUserId, content: $content, metadata: $metadata}' 2>/dev/null) || \
    json="{\"appId\":\"$APP_ID\",\"externalUserId\":\"$USER_ID\",\"content\":$(echo "$content" | jq -Rs .),\"metadata\":{}}"
    local response body http_code
    response=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/seeds" \
        -H "Content-Type: application/json" \
        -d "$json")
    body=$(echo "$response" | head -n -1)
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" = "200" ]; then
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        log_success "Memory saved"
    else
        die "Save failed (HTTP $http_code): $body"
    fi
}

# search - Semantic search (like neutron-memory.sh search "query" [limit] [threshold])
cmd_search() {
    local query="${1:-}"
    local limit="${2:-30}"
    local threshold="${3:-0.5}"
    [ -z "$query" ] && die "Usage: $0 search \"<query>\" [limit] [threshold]"
    local json
    json=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg query "$query" \
        --argjson limit "$limit" \
        --argjson threshold "$threshold" \
        '{appId: $appId, externalUserId: $externalUserId, query: $query, limit: $limit, threshold: $threshold}' 2>/dev/null) || \
    json="{\"appId\":\"$APP_ID\",\"externalUserId\":\"$USER_ID\",\"query\":$(echo "$query" | jq -Rs .),\"limit\":$limit,\"threshold\":$threshold}"
    local response body http_code
    response=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/seeds/query" \
        -H "Content-Type: application/json" \
        -d "$json")
    body=$(echo "$response" | head -n -1)
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" = "200" ]; then
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        die "Search failed (HTTP $http_code): $body"
    fi
}

# context-create - Create agent context (episodic, semantic, procedural, working)
cmd_context_create() {
    local agent_id="${1:-}"
    local memory_type="${2:-episodic}"
    local payload="${3:-{}}"
    [ -z "$agent_id" ] && die "Usage: $0 context-create <agentId> [memoryType] [payload_json]"
    echo "$payload" | jq . >/dev/null 2>&1 || payload="{}"
    local json
    json=$(jq -n \
        --arg appId "$APP_ID" \
        --arg externalUserId "$USER_ID" \
        --arg agentId "$agent_id" \
        --arg memoryType "$memory_type" \
        --argjson payload "$payload" \
        '{appId: $appId, externalUserId: $externalUserId, agentId: $agentId, memoryType: $memoryType, payload: $payload}')
    local response body http_code
    response=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/agent-contexts" \
        -H "Content-Type: application/json" \
        -d "$json")
    body=$(echo "$response" | head -n -1)
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        log_success "Agent context created"
    else
        die "Context create failed (HTTP $http_code): $body"
    fi
}

# context-list - List agent contexts
cmd_context_list() {
    local agent_id="${1:-}"
    local url="${API_URL}/agent-contexts?appId=${APP_ID}&externalUserId=${USER_ID}"
    [ -n "$agent_id" ] && url="${url}&agentId=${agent_id}"
    local response body http_code
    response=$(curl -s -w "\n%{http_code}" "$url")
    body=$(echo "$response" | head -n -1)
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" = "200" ]; then
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        die "Context list failed (HTTP $http_code): $body"
    fi
}

# context-get - Get one agent context by ID
cmd_context_get() {
    local id="${1:-}"
    [ -z "$id" ] && die "Usage: $0 context-get <id>"
    local response body http_code
    response=$(curl -s -w "\n%{http_code}" "${API_URL}/agent-contexts/${id}")
    body=$(echo "$response" | head -n -1)
    http_code=$(echo "$response" | tail -n 1)
    if [ "$http_code" = "200" ]; then
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        die "Context get failed (HTTP $http_code): $body"
    fi
}

# Main
case "${1:-}" in
    test)           cmd_test ;;
    save)           shift; cmd_save "$@" ;;
    search)         shift; cmd_search "$@" ;;
    context-create) shift; cmd_context_create "$@" ;;
    context-list)   shift; cmd_context_list "$@" ;;
    context-get)    shift; cmd_context_get "$@" ;;
    help|--help|-h)
        cat <<EOF
Cortex Memory Script (Neutron-compatible)

Usage: $0 <command> [args...]

Commands:
  test                    Verify Cortex connection (like neutron-memory.sh test)
  save "content" [meta]   Save a memory (metadata optional JSON)
  search "query" [limit] [threshold]   Semantic search (default limit=30, threshold=0.5)
  context-create <agentId> [memoryType] [payload_json]   Create agent context (memoryType: episodic|semantic|procedural|working)
  context-list [agentId]  List agent contexts
  context-get <id>        Get one agent context by ID

Environment (same as OpenClaw Neutron guide where possible):
  CORTEX_API_URL / NEUTRON_API_URL   API base URL (default: http://localhost:9123)
  CORTEX_APP_ID / NEUTRON_AGENT_ID  App/agent ID (default: openclaw)
  CORTEX_USER_ID / YOUR_AGENT_IDENTIFIER   User/identifier (default: default)

Hooks (for OpenClaw integration, when implemented):
  VANAR_AUTO_RECALL   If true, recall relevant memories before each interaction (default: true)
  VANAR_AUTO_CAPTURE  If true, capture conversation after each exchange (default: true)
EOF
        ;;
    *)
        [ -n "${1:-}" ] && log_error "Unknown command: $1"
        echo "Usage: $0 test | save | search | context-create | context-list | context-get | help"
        exit 1
        ;;
esac
