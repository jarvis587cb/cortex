#!/bin/bash
# OpenClaw Hooks: Auto-Recall (before interaction) and Auto-Capture (after exchange).
# Call from OpenClaw or any orchestrator:
#   ./skills/cortex/hooks.sh recall "[query]"     # before AI interaction
#   ./skills/cortex/hooks.sh capture "[content]"   # after exchange
# Respects VANAR_AUTO_RECALL and VANAR_AUTO_CAPTURE (default: true).

set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Path to cortex-memory.sh: from Cortex repo = ../../scripts/cortex-memory.sh; override with CORTEX_MEMORY_SCRIPT
CORTEX_SCRIPT="${CORTEX_MEMORY_SCRIPT:-${HOOKS_DIR}/../../scripts/cortex-memory.sh}"

if [ ! -x "$CORTEX_SCRIPT" ]; then
    echo "Cortex script not found or not executable: $CORTEX_SCRIPT" >&2
    echo "Set CORTEX_MEMORY_SCRIPT to the path of cortex-memory.sh." >&2
    exit 1
fi

exec "$CORTEX_SCRIPT" "$@"
