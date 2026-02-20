#!/bin/bash

# Cortex systemd User Service Installation Script
# Installiert cortex-server als systemd user service

set -e

# Farben für Output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Funktionen
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Prüfe, ob wir im Cortex-Verzeichnis sind
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

if [ ! -f "$PROJECT_ROOT/go.mod" ] || [ ! -d "$PROJECT_ROOT/cmd/cortex-server" ]; then
    error "Bitte führe dieses Script im Cortex-Projektverzeichnis aus!"
    exit 1
fi

info "Cortex systemd User Service Installation wird gestartet..."

# 1. Prüfe, ob Binary existiert
if [ ! -f "$PROJECT_ROOT/cortex-server" ]; then
    warn "Binary cortex-server nicht gefunden. Baue Binary..."
    cd "$PROJECT_ROOT"
    make build-server
    if [ ! -f "$PROJECT_ROOT/cortex-server" ]; then
        error "Binary konnte nicht erstellt werden!"
        exit 1
    fi
    info "✓ Binary erstellt"
fi

# 2. Erstelle systemd user service directory falls nicht vorhanden
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
mkdir -p "$SYSTEMD_USER_DIR"
info "✓ Systemd user directory: $SYSTEMD_USER_DIR"

# 3. Wähle Service-File (Projekt-Binary oder installiertes Binary)
if command -v cortex-server &> /dev/null && [ "$(which cortex-server)" != "$PROJECT_ROOT/cortex-server" ]; then
    info "Installiertes Binary gefunden. Verwende cortex-server-installed.service"
    SERVICE_FILE="$SCRIPT_DIR/cortex-server-installed.service"
else
    info "Verwende Binary aus Projekt-Verzeichnis"
    SERVICE_FILE="$SCRIPT_DIR/cortex-server.service"
fi

if [ ! -f "$SERVICE_FILE" ]; then
    error "Service-File nicht gefunden: $SERVICE_FILE"
    exit 1
fi

# Ersetze %h durch $HOME im Service-File
TEMP_SERVICE=$(mktemp)
sed "s|%h|$HOME|g" "$SERVICE_FILE" > "$TEMP_SERVICE"

# Prüfe, ob Service bereits existiert
if [ -f "$SYSTEMD_USER_DIR/cortex-server.service" ]; then
    warn "Service-File existiert bereits. Überschreibe..."
    read -p "Fortfahren? (j/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[JjYy]$ ]]; then
        rm "$TEMP_SERVICE"
        info "Abgebrochen."
        exit 0
    fi
fi

cp "$TEMP_SERVICE" "$SYSTEMD_USER_DIR/cortex-server.service"
rm "$TEMP_SERVICE"
info "✓ Service-File kopiert: $SYSTEMD_USER_DIR/cortex-server.service"

# 4. Reload systemd
info "Lade systemd user daemon neu..."
systemctl --user daemon-reload
info "✓ Systemd neu geladen"

# 5. Aktiviere Service (startet nicht automatisch, nur beim Login)
info "Aktiviere Service..."
systemctl --user enable cortex-server.service
info "✓ Service aktiviert"

# 6. Starte Service
info "Starte Service..."
systemctl --user start cortex-server.service
info "✓ Service gestartet"

# 7. Zeige Status
echo ""
info "=========================================="
info "Installation abgeschlossen!"
info "=========================================="
echo ""
info "Service-Status:"
systemctl --user status cortex-server.service --no-pager || true
echo ""
info "Nützliche Befehle:"
echo ""
echo "  Status prüfen:"
echo "    ${YELLOW}systemctl --user status cortex-server${NC}"
echo ""
echo "  Logs anzeigen:"
echo "    ${YELLOW}journalctl --user -u cortex-server -f${NC}"
echo ""
echo "  Service stoppen:"
echo "    ${YELLOW}systemctl --user stop cortex-server${NC}"
echo ""
echo "  Service starten:"
echo "    ${YELLOW}systemctl --user start cortex-server${NC}"
echo ""
echo "  Service neu starten:"
echo "    ${YELLOW}systemctl --user restart cortex-server${NC}"
echo ""
echo "  Service deaktivieren:"
echo "    ${YELLOW}systemctl --user disable cortex-server${NC}"
echo ""
info "Der Service startet automatisch beim Login."
echo ""
