#!/bin/bash

# Cortex Setup Script
# Erstellt Konfigurationsdateien und richtet Cortex ein

set -e

# Farben für Output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Funktionen
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

question() {
    echo -e "${BLUE}[?]${NC} $1"
}

# Prüfe, ob wir im Cortex-Verzeichnis sind
if [ ! -f "go.mod" ] || [ ! -d "cmd/cortex-server" ]; then
    echo "Bitte führe dieses Script im Cortex-Projektverzeichnis aus!"
    exit 1
fi

info "Cortex Setup wird gestartet..."

# 1. Erstelle ~/.openclaw Verzeichnis
info "Erstelle Verzeichnisse..."
mkdir -p ~/.openclaw
info "✓ Verzeichnis erstellt: ~/.openclaw"

# 2. Erstelle Config-Datei (falls nicht vorhanden)
CONFIG_FILE="$HOME/.openclaw/cortex.json"

if [ -f "$CONFIG_FILE" ]; then
    warn "Config-Datei existiert bereits: $CONFIG_FILE"
    question "Möchtest du sie überschreiben? (j/n)"
    read -r response
    if [ "$response" != "j" ] && [ "$response" != "J" ]; then
        info "Setup abgebrochen - Config-Datei bleibt unverändert"
        exit 0
    fi
fi

info "Erstelle Config-Datei: $CONFIG_FILE"

# Frage nach Konfigurationswerten
question "Port für Cortex-Server (Standard: 9123):"
read -r port
port=${port:-9123}

question "Rate Limit (Standard: 100):"
read -r rate_limit
rate_limit=${rate_limit:-100}

# Erstelle JSON-Config
cat > "$CONFIG_FILE" << EOF
{
  "db_path": "$HOME/.openclaw/cortex.db",
  "port": $port,
  "log_level": "info",
  "rate_limit": $rate_limit,
  "rate_limit_window": "1m"
}
EOF

info "✓ Config-Datei erstellt: $CONFIG_FILE"

# 3. Erstelle .env Datei (optional)
ENV_FILE=".env"

if [ -f "$ENV_FILE" ]; then
    warn ".env Datei existiert bereits"
    question "Möchtest du sie aktualisieren? (j/n)"
    read -r response
    if [ "$response" != "j" ] && [ "$response" != "J" ]; then
        info ".env Datei bleibt unverändert"
    else
        CREATE_ENV=true
    fi
else
    CREATE_ENV=true
fi

if [ "$CREATE_ENV" = true ]; then
    info "Erstelle .env Datei..."
    cat > "$ENV_FILE" << EOF
# Cortex Configuration
CORTEX_DB_PATH=$HOME/.openclaw/cortex.db
CORTEX_PORT=$port
CORTEX_LOG_LEVEL=info
CORTEX_RATE_LIMIT=$rate_limit
CORTEX_RATE_LIMIT_WINDOW=1m
EOF
    info "✓ .env Datei erstellt"
fi

# 4. Erstelle systemd Service-Datei (optional)
question "Möchtest du einen systemd Service erstellen? (j/n)"
read -r create_service

if [ "$create_service" = "j" ] || [ "$create_service" = "J" ]; then
    SERVICE_FILE="$HOME/.config/systemd/user/cortex.service"
    SERVICE_DIR=$(dirname "$SERVICE_FILE")
    
    mkdir -p "$SERVICE_DIR"
    
    CORTEX_BINARY=$(pwd)/cortex-server
    
    if [ ! -f "$CORTEX_BINARY" ]; then
        warn "Binary nicht gefunden: $CORTEX_BINARY"
        question "Möchtest du trotzdem die Service-Datei erstellen? (j/n)"
        read -r response
        if [ "$response" != "j" ] && [ "$response" != "J" ]; then
            info "Service-Erstellung übersprungen"
        else
            CREATE_SERVICE=true
        fi
    else
        CREATE_SERVICE=true
    fi
    
    if [ "$CREATE_SERVICE" = true ]; then
        info "Erstelle systemd Service-Datei: $SERVICE_FILE"
        
        cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Cortex Memory API Server
After=network.target

[Service]
Type=simple
ExecStart=$CORTEX_BINARY
WorkingDirectory=$(pwd)
Restart=always
RestartSec=5
Environment="CORTEX_DB_PATH=$HOME/.openclaw/cortex.db"
Environment="CORTEX_PORT=$port"
Environment="CORTEX_LOG_LEVEL=info"
EOF
        
        info "✓ Service-Datei erstellt: $SERVICE_FILE"
        info "  Zum Aktivieren: systemctl --user enable cortex"
        info "  Zum Starten: systemctl --user start cortex"
    fi
fi

# 5. Erstelle Shell-Aliase (optional)
question "Möchtest du Shell-Aliase erstellen? (j/n)"
read -r create_aliases

if [ "$create_aliases" = "j" ] || [ "$create_aliases" = "J" ]; then
    SHELL_RC=""
    
    if [ -f "$HOME/.bashrc" ]; then
        SHELL_RC="$HOME/.bashrc"
    elif [ -f "$HOME/.zshrc" ]; then
        SHELL_RC="$HOME/.zshrc"
    fi
    
    if [ -n "$SHELL_RC" ]; then
        CORTEX_DIR=$(pwd)
        
        if ! grep -q "Cortex Aliases" "$SHELL_RC"; then
            info "Füge Aliase zu $SHELL_RC hinzu..."
            cat >> "$SHELL_RC" << EOF

# Cortex Aliases
alias cortex-start='cd $CORTEX_DIR && ./cortex-server'
alias cortex-test='cd $CORTEX_DIR && go test ./...'
alias cortex-build='cd $CORTEX_DIR && make build'
alias cortex-health='curl http://localhost:$port/health'
EOF
            info "✓ Aliase hinzugefügt"
            info "  Starte eine neue Shell-Session oder führe aus: source $SHELL_RC"
        else
            warn "Aliase existieren bereits in $SHELL_RC"
        fi
    else
        warn "Keine .bashrc oder .zshrc gefunden - Aliase werden nicht erstellt"
    fi
fi

# Zusammenfassung
echo ""
info "=========================================="
info "Setup abgeschlossen!"
info "=========================================="
echo ""
info "Konfiguration:"
echo "  - Config-Datei: $CONFIG_FILE"
echo "  - Datenbank: $HOME/.openclaw/cortex.db"
echo "  - Port: $port"
echo ""
info "Nächste Schritte:"
echo ""
echo "  1. Starte den Server:"
echo "     ${YELLOW}./cortex-server${NC}"
echo "     oder"
echo "     ${YELLOW}go run ./cmd/cortex-server${NC}"
echo ""
echo "  2. Prüfe den Health-Check:"
echo "     ${YELLOW}curl http://localhost:$port/health${NC}"
echo ""
echo "  3. Teste die API:"
echo "     ${YELLOW}curl -X POST http://localhost:$port/seeds?appId=test&externalUserId=user1 \\"
echo "       -H 'Content-Type: application/json' \\"
echo "       -d '{\"content\": \"Test Memory\"}'${NC}"
echo ""
info "Dokumentation: Siehe README.md und API.md"
echo ""
