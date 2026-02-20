#!/bin/bash
# benchmark-embeddings.sh - Wrapper für Embedding-Benchmark mit Positionsargumenten
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse Arguments
COUNT="${1:-100}"
SERVICE="${2:-both}"

# Validate COUNT
if ! [[ "$COUNT" =~ ^[0-9]+$ ]] || [ "$COUNT" -lt 1 ]; then
    echo "Fehler: COUNT muss eine positive Ganzzahl sein (gegeben: $COUNT)"
    exit 1
fi

# Validate SERVICE
if [[ "$SERVICE" != "local" && "$SERVICE" != "gte" && "$SERVICE" != "both" ]]; then
    echo "Fehler: SERVICE muss 'local', 'gte' oder 'both' sein (gegeben: $SERVICE)"
    exit 1
fi

# Ensure CLI is built
if [ ! -f "$PROJECT_ROOT/cortex-cli" ]; then
    echo "Baue cortex-cli..."
    cd "$PROJECT_ROOT"
    go build -o cortex-cli ./cmd/cortex-cli
fi

# Run benchmark
"$PROJECT_ROOT/cortex-cli" benchmark-embeddings "$COUNT" "$SERVICE"
