#!/bin/bash

# Cortex Installation Script
# Installiert Cortex und alle Dependencies

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
if [ ! -f "go.mod" ] || [ ! -d "cmd/cortex-server" ]; then
    error "Bitte führe dieses Script im Cortex-Projektverzeichnis aus!"
    exit 1
fi

info "Cortex Installation wird gestartet..."

# 1. Prüfe Go-Installation
info "Prüfe Go-Installation..."
if ! command -v go &> /dev/null; then
    error "Go ist nicht installiert! Bitte installiere Go 1.23+ von https://go.dev"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
info "Go Version: $GO_VERSION"

# Prüfe Go-Version (mindestens 1.23)
if [ "$(printf '%s\n' "1.23" "$GO_VERSION" | sort -V | head -n1)" != "1.23" ]; then
    warn "Go Version sollte mindestens 1.23 sein. Aktuelle Version: $GO_VERSION"
fi

# 2. Installiere Dependencies
info "Installiere Go-Dependencies..."
go mod download
go mod tidy

if [ $? -ne 0 ]; then
    error "Fehler beim Installieren der Dependencies!"
    exit 1
fi

info "✓ Dependencies installiert"

# 3. Baue Binary
info "Baue Cortex Binary..."
make build

if [ $? -ne 0 ]; then
    error "Fehler beim Bauen der Binary!"
    exit 1
fi

info "✓ Binaries erstellt: ./cortex-server, ./cortex-cli"

# 4. Führe Tests aus
info "Führe Tests aus..."
go test ./... -short

if [ $? -ne 0 ]; then
    warn "Einige Tests sind fehlgeschlagen. Bitte prüfe die Ausgabe."
else
    info "✓ Alle Tests bestanden"
fi

# 5. Erstelle Verzeichnisse
info "Erstelle benötigte Verzeichnisse..."
mkdir -p ~/.openclaw
info "✓ Verzeichnis erstellt: ~/.openclaw"

# 6. Prüfe optional: Make
if command -v make &> /dev/null; then
    info "✓ Make ist verfügbar - Makefile-Targets können verwendet werden"
else
    warn "Make ist nicht installiert - Makefile-Targets sind nicht verfügbar"
fi

# 7. Prüfe optional: Docker
if command -v docker &> /dev/null; then
    info "✓ Docker ist verfügbar - Docker-Build ist möglich"
else
    warn "Docker ist nicht installiert - Docker-Build ist nicht möglich"
fi

# Zusammenfassung
echo ""
info "=========================================="
info "Installation abgeschlossen!"
info "=========================================="
echo ""
info "Nächste Schritte:"
echo ""
echo "  1. Starte den Server:"
echo "     ${YELLOW}./cortex-server${NC}"
echo "     oder"
echo "     ${YELLOW}go run ./cmd/cortex-server${NC}"
echo ""
echo "  2. Prüfe den Health-Check:"
echo "     ${YELLOW}curl http://localhost:9123/health${NC}"
echo ""
echo "  3. Konfiguriere Umgebungsvariablen (optional):"
echo "     ${YELLOW}export CORTEX_PORT=9123${NC}"
echo ""
echo "  4. Führe das Setup-Script aus:"
echo "     ${YELLOW}./skills/cortex/setup.sh${NC}"
echo ""
info "Dokumentation: Siehe README.md und API.md"
echo ""
