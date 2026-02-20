# Cortex Skill - Installationsanweisungen

Dieser Ordner enthält die Dokumentation und Service-Dateien für Cortex.

## Dateien

- **`SKILL.md`** – Vollständige Dokumentation (Installation, API, CLI, OpenClaw-Integration)
- **`hooks.sh`** – Auto-Recall/Capture Hooks für OpenClaw
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

## Hooks-Konfiguration (OpenClaw-Integration)

Cortex unterstützt automatisches Abrufen (Recall) und Speichern (Capture) von Memories über Skill-Hooks.

### Hooks aktivieren

Die Hooks (`hooks.sh`) sind bereits im Skill enthalten. Konfigurieren Sie sie über Umgebungsvariablen:

```bash
# In .env oder Umgebungsvariablen
CORTEX_AUTO_RECALL=true      # Auto-Recall aktivieren (Default: true)
CORTEX_AUTO_CAPTURE=true     # Auto-Capture aktivieren (Default: true)
CORTEX_API_URL=http://localhost:9123
CORTEX_APP_ID=openclaw
CORTEX_USER_ID=default
```

### Hooks testen

```bash
# Recall-Hook testen
echo '{"message": "Was weißt du über Kaffee?"}' | ./skills/cortex/hooks.sh recall

# Capture-Hook testen
cat <<EOF | ./skills/cortex/hooks.sh capture
{
  "content": "User: Hello\nAI: Hi there!",
  "appId": "openclaw",
  "userId": "user123"
}
EOF
```

### Hooks deaktivieren

```bash
# In .env
CORTEX_AUTO_RECALL=false
CORTEX_AUTO_CAPTURE=false
```

Siehe `SKILL.md` für vollständige Hook-Dokumentation.

## Weitere Informationen

Siehe `SKILL.md` für vollständige Dokumentation.
