# Cortex Skill - Installationsanweisungen

Dieser Ordner enthält Installations- und Setup-Scripte für Cortex.

## Dateien

- **`SKILL.md`** – Hauptdokumentation (Server, API, OpenClaw-Script, Hooks)
- **`install.sh`** – Installations-Script (Dependencies, Build, Tests)
- **`setup.sh`** – Setup-Script (Config, Aliase, systemd Service)
- **`hooks.sh`** – OpenClaw Hooks (Recall/Capture), ruft `scripts/cortex-memory.sh` auf

## Verwendung

### 1. Installation

```bash
# Aus dem Cortex-Projektverzeichnis
./skills/cortex/install.sh
```

Das Script:
- Prüft Go-Installation
- Installiert Dependencies
- Baut die Binary
- Führt Tests aus
- Erstellt benötigte Verzeichnisse

### 2. Setup

```bash
# Nach der Installation
./skills/cortex/setup.sh
```

Das Script:
- Erstellt Config-Datei (`~/.openclaw/cortex.json`)
- Erstellt `.env` Datei
- Optional: systemd Service
- Optional: Shell-Aliase

## Schnellstart

```bash
# 1. Installation
./skills/cortex/install.sh

# 2. Setup
./skills/cortex/setup.sh

# 3. Server starten
./cortex-server
# oder
go run ./cmd/cortex-server

# 4. Health-Check
curl http://localhost:9123/health
```

## Weitere Informationen

Siehe `SKILL.md` für vollständige Dokumentation.
