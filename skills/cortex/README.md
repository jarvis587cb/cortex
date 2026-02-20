# Cortex Skill - Installationsanweisungen

Dieser Ordner enthält die Dokumentation und Service-Dateien für Cortex.

## Dateien

- **`SKILL.md`** – Vollständige Dokumentation (Installation, API, CLI, OpenClaw-Integration)
- **`cortex-server.service`** – systemd User Service-Datei für den Cortex-Server
- **`cortex-server-installed.service`** – Alternative Service-Datei für installierte Binaries

## Installation

### Vollständige Installation (Empfohlen)

```bash
# 1. Repository klonen
git clone https://github.com/jarvis587cb/cortex.git
cd cortex

# 2. Binaries bauen
make build    # Erstellt cortex-server und cortex-cli

# 3. systemd User Service erstellen
mkdir -p ~/.config/systemd/user
cp skills/cortex/cortex-server.service ~/.config/systemd/user/cortex-server.service
# %h durch $HOME ersetzen (falls nötig)
sed -i "s|%h|$HOME|g" ~/.config/systemd/user/cortex-server.service

# 4. Service aktivieren und starten
systemctl --user daemon-reload
systemctl --user enable cortex-server.service
systemctl --user start cortex-server.service

# Status prüfen
systemctl --user status cortex-server
```

### Schnellstart (Manuell ohne Service)

```bash
# 1. Repository klonen und bauen
git clone https://github.com/jarvis587cb/cortex.git
cd cortex
make build

# 2. Server starten
./cortex-server
# oder
go run ./cmd/cortex-server

# 3. Health-Check
curl http://localhost:9123/health
# oder
./cortex-cli health
```

## Weitere Informationen

Siehe `SKILL.md` für vollständige Dokumentation.
