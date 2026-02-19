#!/bin/bash

# Common functions library for Cortex scripts
# Source this file at the beginning of scripts: source "$(dirname "$0")/lib/common.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Fatal error - exit script
die() {
    log_error "$1"
    exit 1
}

# Execute curl and return body and status code
# Usage: result=$(curl_with_status url [curl-args...])
#        body=$(echo "$result" | head -n -1)
#        status=$(echo "$result" | tail -n 1)
curl_with_status() {
    local url="$1"
    shift
    local response
    response=$(curl -s -w "\n%{http_code}" "$@" "$url" 2>/dev/null || echo -e "\n000")
    echo "$response"
}

# Parse HTTP response into body and status code
# Usage: parse_http_response response -> outputs "body\nstatus"
parse_http_response() {
    local response="$1"
    local body=$(echo "$response" | head -n -1)
    local status_code=$(echo "$response" | tail -n 1)
    echo "$body"
    echo "$status_code"
}

# Validation functions
is_positive_integer() {
    local value="$1"
    [[ "$value" =~ ^[0-9]+$ ]] && [ "$value" -gt 0 ]
}

is_empty_or_whitespace() {
    local value="$1"
    [ -z "$value" ] || [ -z "${value// }" ]
}

require_file() {
    local file="$1"
    if [ ! -f "$file" ]; then
        die "Datei nicht gefunden: $file"
    fi
}

# jq helper functions
has_jq() {
    command -v jq &> /dev/null
}

safe_jq() {
    if has_jq; then
        jq "$@"
    else
        die "jq ist nicht installiert. Bitte installiere jq für diese Funktion."
    fi
}

# Format JSON (with jq if available, otherwise cat)
format_json() {
    if has_jq; then
        jq .
    else
        cat
    fi
}

# Extract ID from JSON response
# Usage: extract_id json_body -> outputs id or empty string
extract_id() {
    local json_body="$1"
    if has_jq; then
        echo "$json_body" | jq -r '.id // empty' 2>/dev/null || echo ""
    else
        echo "$json_body" | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1 || echo ""
    fi
}

# Count items in JSON array
# Usage: count_items json_array -> outputs count
count_items() {
    local json_array="$1"
    if has_jq; then
        echo "$json_array" | jq 'length' 2>/dev/null || echo "0"
    else
        echo "$json_array" | grep -o '"id"' | wc -l || echo "0"
    fi
}
