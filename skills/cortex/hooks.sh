#!/bin/bash
# OpenClaw Hooks: Auto-Recall (before interaction) and Auto-Capture (after exchange).
# Call from OpenClaw or any orchestrator:
#   ./skills/cortex/hooks.sh recall "[query]"     # before AI interaction
#   ./skills/cortex/hooks.sh capture "[content]"   # after exchange
# Respects CORTEX_AUTO_RECALL and CORTEX_AUTO_CAPTURE (default: true).

set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Path to cortex-cli: from Cortex repo = ../../cortex-cli; override with CORTEX_CLI_PATH
CORTEX_CLI="${CORTEX_CLI_PATH:-${HOOKS_DIR}/../../cortex-cli}"

if [ ! -x "$CORTEX_CLI" ]; then
    echo "cortex-cli not found or not executable: $CORTEX_CLI" >&2
    echo "Set CORTEX_CLI_PATH to the path of cortex-cli or build it with: make build-cli" >&2
    exit 1
fi

# Auto-Recall/Capture können deaktiviert werden
AUTO_RECALL="${CORTEX_AUTO_RECALL:-true}"
AUTO_CAPTURE="${CORTEX_AUTO_CAPTURE:-true}"

cmd="${1:-}"
shift || true

case "$cmd" in
    recall)
        if [ "$AUTO_RECALL" != "true" ] && [ "$AUTO_RECALL" != "1" ]; then
            exit 0
        fi
        query="${1:-}"
        if [ -z "$query" ]; then
            echo "Usage: $0 recall \"<query>\"" >&2
            exit 1
        fi
        # Suche nach relevanten Memories
        "$CORTEX_CLI" query "$query" 10 0.2
        ;;
    capture)
        if [ "$AUTO_CAPTURE" != "true" ] && [ "$AUTO_CAPTURE" != "1" ]; then
            exit 0
        fi
        content="${1:-}"
        if [ -z "$content" ]; then
            echo "Usage: $0 capture \"<content>\"" >&2
            exit 1
        fi
        # Speichere Memory
        "$CORTEX_CLI" store "$content" '{"source":"hook","timestamp":"'$(date -Iseconds)'"}'
        ;;
    *)
        echo "Usage: $0 <recall|capture> [args...]" >&2
        echo "  recall <query>   - Suche nach relevanten Memories" >&2
        echo "  capture <content> - Speichere Memory" >&2
        exit 1
        ;;
esac
