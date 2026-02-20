#!/bin/bash
# test-hooks.sh - Test script for Cortex hooks
# Tests recall and capture hooks with example inputs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOKS_SCRIPT="$SCRIPT_DIR/hooks.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if hooks.sh exists
if [ ! -f "$HOOKS_SCRIPT" ]; then
    echo_error "hooks.sh not found at $HOOKS_SCRIPT"
    exit 1
fi

# Check if hooks.sh is executable
if [ ! -x "$HOOKS_SCRIPT" ]; then
    echo_warn "hooks.sh is not executable, making it executable..."
    chmod +x "$HOOKS_SCRIPT"
fi

# Check if cortex-cli is available
if ! command -v cortex-cli >/dev/null 2>&1 && [ ! -f "$SCRIPT_DIR/../../cortex-cli" ]; then
    echo_warn "cortex-cli not found. Some tests may fail."
    echo_warn "Make sure cortex-cli is in PATH or build it with 'make build'"
fi

echo_info "Testing Cortex Hooks"
echo_info "===================="
echo ""

# Test 1: Recall hook with JSON input
echo_info "Test 1: Recall hook with JSON input"
echo '{"message": "What do you know about coffee?", "appId": "test-app", "userId": "test-user"}' | "$HOOKS_SCRIPT" recall
echo ""

# Test 2: Recall hook with plain text input
echo_info "Test 2: Recall hook with plain text input"
echo "What do you know about coffee?" | "$HOOKS_SCRIPT" recall
echo ""

# Test 3: Recall hook disabled
echo_info "Test 3: Recall hook disabled (should return empty array)"
CORTEX_AUTO_RECALL=false echo "test query" | "$HOOKS_SCRIPT" recall
echo ""

# Test 4: Capture hook with JSON input
echo_info "Test 4: Capture hook with JSON input"
cat <<EOF | "$HOOKS_SCRIPT" capture
{
  "content": "User: What is Cortex?\nAI: Cortex is a local memory system for OpenClaw agents.",
  "appId": "test-app",
  "userId": "test-user",
  "metadata": {
    "platform": "test",
    "source": "test-hooks.sh"
  }
}
EOF
echo ""

# Test 5: Capture hook disabled
echo_info "Test 5: Capture hook disabled (should do nothing)"
CORTEX_AUTO_CAPTURE=false cat <<EOF | "$HOOKS_SCRIPT" capture
{
  "content": "This should not be stored",
  "appId": "test-app",
  "userId": "test-user"
}
EOF
echo ""

# Test 6: Invalid command
echo_info "Test 6: Invalid command (should show usage)"
"$HOOKS_SCRIPT" invalid-command 2>&1 || true
echo ""

echo_info "Tests completed!"
echo_info "Note: Make sure cortex-server is running for full functionality"
echo_info "Start server with: make run"
