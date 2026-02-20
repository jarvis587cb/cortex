#!/bin/bash
# hooks.sh - Cortex skill hooks for OpenClaw
# Auto-Recall/Capture hooks for automatic memory retrieval and storage
# Usage: hooks.sh {recall|capture} [options]

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Find cortex-cli binary
find_cortex_cli() {
    if command -v cortex-cli >/dev/null 2>&1; then
        echo "cortex-cli"
    elif [ -f "$PROJECT_ROOT/cortex-cli" ]; then
        echo "$PROJECT_ROOT/cortex-cli"
    elif [ -f "./cortex-cli" ]; then
        echo "./cortex-cli"
    else
        echo "cortex-cli" >&2
        return 1
    fi
}

CORTEX_CLI=$(find_cortex_cli 2>/dev/null || echo "cortex-cli")

# Configuration from environment variables
CORTEX_AUTO_RECALL="${CORTEX_AUTO_RECALL:-true}"
CORTEX_AUTO_CAPTURE="${CORTEX_AUTO_CAPTURE:-true}"
CORTEX_API_URL="${CORTEX_API_URL:-http://localhost:9123}"
CORTEX_APP_ID="${CORTEX_APP_ID:-openclaw}"
CORTEX_USER_ID="${CORTEX_USER_ID:-default}"
CORTEX_API_KEY="${CORTEX_API_KEY:-}"

# Default query parameters
RECALL_LIMIT="${CORTEX_RECALL_LIMIT:-5}"
RECALL_THRESHOLD="${CORTEX_RECALL_THRESHOLD:-0.5}"

# Helper function to parse JSON input (simple, no jq dependency)
parse_json_field() {
    local json="$1"
    local field="$2"
    # Simple JSON parsing: extract value for field
    echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | sed "s/\"$field\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\"/\1/" || echo ""
}

# Helper function to check if string is JSON
is_json() {
    local str="$1"
    [[ "$str" =~ ^[[:space:]]*\{ ]] || [[ "$str" =~ ^[[:space:]]*\[ ]]
}

# Recall hook: Retrieve relevant memories before AI interaction
recall_hook() {
    if [ "$CORTEX_AUTO_RECALL" != "true" ]; then
        echo "[]"
        return 0
    fi

    # Read input from stdin
    local input=""
    if [ -t 0 ]; then
        # No stdin, try to read from arguments
        input="${1:-}"
    else
        # Read from stdin
        input=$(cat)
    fi

    if [ -z "$input" ]; then
        echo "[]" >&2
        echo "[]"
        return 0
    fi

    # Extract query text and tenant info
    local query=""
    local app_id="$CORTEX_APP_ID"
    local user_id="$CORTEX_USER_ID"

    if is_json "$input"; then
        # JSON input: extract message, appId, userId
        query=$(parse_json_field "$input" "message" || echo "")
        if [ -z "$query" ]; then
            # Try "content" or "text" fields
            query=$(parse_json_field "$input" "content" || parse_json_field "$input" "text" || echo "")
        fi
        
        local extracted_app_id=$(parse_json_field "$input" "appId" || echo "")
        local extracted_user_id=$(parse_json_field "$input" "userId" || echo "")
        
        [ -n "$extracted_app_id" ] && app_id="$extracted_app_id"
        [ -n "$extracted_user_id" ] && user_id="$extracted_user_id"
    else
        # Plain text input
        query="$input"
    fi

    if [ -z "$query" ]; then
        echo "[]"
        return 0
    fi

    # Call cortex-cli query
    local env_vars=""
    [ -n "$CORTEX_API_URL" ] && env_vars="CORTEX_API_URL=\"$CORTEX_API_URL\" "
    [ -n "$CORTEX_APP_ID" ] && env_vars="${env_vars}CORTEX_APP_ID=\"$app_id\" "
    [ -n "$CORTEX_USER_ID" ] && env_vars="${env_vars}CORTEX_USER_ID=\"$user_id\" "
    [ -n "$CORTEX_API_KEY" ] && env_vars="${env_vars}CORTEX_API_KEY=\"$CORTEX_API_KEY\" "

    # Execute query and capture output
    local output=""
    if output=$(eval "$env_vars $CORTEX_CLI query \"$query\" $RECALL_LIMIT $RECALL_THRESHOLD" 2>&1); then
        # Parse cortex-cli output (it outputs JSON array)
        if is_json "$output"; then
            echo "$output"
        else
            # If output is not JSON, return empty array
            echo "[]"
        fi
    else
        # On error, return empty array (don't break agent flow)
        echo "Error in recall hook: $output" >&2
        echo "[]"
        return 0
    fi
}

# Capture hook: Store conversation after exchange
capture_hook() {
    if [ "$CORTEX_AUTO_CAPTURE" != "true" ]; then
        return 0
    fi

    # Read input from stdin
    local input=""
    if [ -t 0 ]; then
        # No stdin, try to read from arguments
        input="${1:-}"
    else
        # Read from stdin
        input=$(cat)
    fi

    if [ -z "$input" ]; then
        echo "No input provided to capture hook" >&2
        return 1
    fi

    # Extract content and tenant info
    local content=""
    local app_id="$CORTEX_APP_ID"
    local user_id="$CORTEX_USER_ID"
    local metadata="{}"

    if is_json "$input"; then
        # JSON input: extract content, appId, userId, metadata
        content=$(parse_json_field "$input" "content" || echo "")
        
        local extracted_app_id=$(parse_json_field "$input" "appId" || echo "")
        local extracted_user_id=$(parse_json_field "$input" "userId" || echo "")
        
        [ -n "$extracted_app_id" ] && app_id="$extracted_app_id"
        [ -n "$extracted_user_id" ] && user_id="$extracted_user_id"
        
        # Try to extract metadata (simplified - just use the whole JSON if it has metadata field)
        if echo "$input" | grep -q "\"metadata\""; then
            # For simplicity, we'll pass the whole input as metadata context
            metadata="$input"
        fi
    else
        # Plain text input
        content="$input"
    fi

    if [ -z "$content" ]; then
        echo "No content found in input" >&2
        return 1
    fi

    # Add automatic metadata
    local auto_metadata="{\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"hook_version\":\"1.0\",\"source\":\"openclaw_hook\"}"
    if [ "$metadata" != "{}" ]; then
        # Merge metadata (simplified)
        metadata="$auto_metadata"
    else
        metadata="$auto_metadata"
    fi

    # Call cortex-cli store
    local env_vars=""
    [ -n "$CORTEX_API_URL" ] && env_vars="CORTEX_API_URL=\"$CORTEX_API_URL\" "
    [ -n "$CORTEX_APP_ID" ] && env_vars="${env_vars}CORTEX_APP_ID=\"$app_id\" "
    [ -n "$CORTEX_USER_ID" ] && env_vars="${env_vars}CORTEX_USER_ID=\"$user_id\" "
    [ -n "$CORTEX_API_KEY" ] && env_vars="${env_vars}CORTEX_API_KEY=\"$CORTEX_API_KEY\" "

    # Execute store command
    if eval "$env_vars $CORTEX_CLI store \"$content\" '$metadata'" >/dev/null 2>&1; then
        return 0
    else
        echo "Error storing memory in capture hook" >&2
        return 1
    fi
}

# Main command dispatcher
case "${1:-}" in
    recall)
        recall_hook "${@:2}"
        ;;
    capture)
        capture_hook "${@:2}"
        ;;
    *)
        echo "Usage: $0 {recall|capture}" >&2
        echo "" >&2
        echo "Commands:" >&2
        echo "  recall   - Retrieve relevant memories before AI interaction" >&2
        echo "  capture  - Store conversation after exchange" >&2
        echo "" >&2
        echo "Environment variables:" >&2
        echo "  CORTEX_AUTO_RECALL     - Enable/disable recall (default: true)" >&2
        echo "  CORTEX_AUTO_CAPTURE    - Enable/disable capture (default: true)" >&2
        echo "  CORTEX_API_URL         - Cortex API URL (default: http://localhost:9123)" >&2
        echo "  CORTEX_APP_ID          - Application ID (default: openclaw)" >&2
        echo "  CORTEX_USER_ID         - User ID (default: default)" >&2
        echo "  CORTEX_API_KEY         - Optional API key" >&2
        echo "  CORTEX_RECALL_LIMIT    - Max results for recall (default: 5)" >&2
        echo "  CORTEX_RECALL_THRESHOLD - Similarity threshold (default: 0.5)" >&2
        exit 1
        ;;
esac
