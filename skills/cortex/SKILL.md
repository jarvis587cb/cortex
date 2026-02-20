---
name: cortex
description: "Vollständig lokale, persistente Memory-API für OpenClaw Agents. Server-Installation, API-Nutzung und OpenClaw-Integration mit Neutron-kompatiblen CLI-Befehlen."
metadata:
  {
    "openclaw":
      {
        "emoji": "🧠",
        "requires": { "bins": ["cortex-server", "cortex-cli"] },
        "install":
          [
            {
              "id": "build",
              "kind": "script",
              "script": "cd /path/to/cortex && make build",
              "bins": ["cortex-server", "cortex-cli"],
              "label": "Build Cortex (make build)",
            },
          ],
      },
  }
---

# Cortex Skill

**Cortex** ist eine **vollständig lokale**, persistente Memory-API für OpenClaw Agents. Ein Skill für Server-Installation, API-Nutzung und OpenClaw-Integration mit Neutron-kompatiblen CLI-Befehlen.

## References

- `README.md` – Projekt-Überblick, Build, Docker, Architektur
- `docs/API.md` – Vollständige API-Dokumentation
- `docs/CORTEX_NEUTRON_ALTERNATIVE.md` – Neutron-Alternative, OpenClaw-Guide-Vergleich
- `docs/VERGLEICH_NEUTRON.md` – Feature-für-Feature-Vergleich mit Neutron
- `scripts/README.md` – CLI-Tool Dokumentation

## Was ist Cortex?

Cortex ist ein **leichtgewichtiges Go-Backend** mit SQLite-Datenbank, das als persistentes "Gehirn" für OpenClaw-Agenten dient. Es speichert Erinnerungen (Memories), Entities mit Fakten sowie Relationen zwischen Entities.

### Hauptfeatures

- **Persistente Speicherung**: SQLite-Datenbank überlebt Neustarts
- **Semantische Suche**: Vektor-basierte Suche mit lokalen Embeddings
- **Multi-Tenant**: Isolation durch `appId` + `externalUserId`
- **Neutron-kompatibel**: Gleiche API-Formate wie Neutron; Script-Befehle wie im [OpenClaw Integration Guide](https://openclaw.vanarchain.com/guide-openclaw)
- **Bundles**: Organisation von Memories in logische Gruppen
- **Webhooks, Analytics, Export/Import, Backup/Restore**

---

## Prerequisites

1. **Go 1.23+** installiert (`go version` zum Prüfen)
2. **Git** und optional **Make**
3. Cortex-Projekt geklont oder vorhanden
4. (Optional) Docker für Container-Deployment

---

## Installation

### Vollständige Installation (Empfohlen)

**Schritt-für-Schritt-Anleitung:**

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
cd /path/to/cortex
go mod tidy
make build    # Erstellt cortex-server und cortex-cli
make run      # Startet den Server
```

### Docker

```bash
make docker-build
make docker-run
# oder: docker-compose up -d
```

### Binaries installieren

```bash
make install  # Installiert beide Binaries nach /usr/local/bin
```

### systemd User Service (Empfohlen für dauerhaften Betrieb)

Installiere cortex-server als systemd user service, der automatisch beim Login startet:

**Manuelle Installation:**

```bash
# 1. Service-File kopieren
mkdir -p ~/.config/systemd/user
cp skills/cortex/cortex-server.service ~/.config/systemd/user/cortex-server.service

# 2. %h durch $HOME ersetzen (falls nötig)
sed -i "s|%h|$HOME|g" ~/.config/systemd/user/cortex-server.service

# 3. Service aktivieren und starten
systemctl --user daemon-reload
systemctl --user enable cortex-server.service
systemctl --user start cortex-server.service
```

Der Service:
- Startet automatisch beim Login
- Startet neu bei Fehlern (Restart=always)
- Loggt in systemd journal (`journalctl --user -u cortex-server`)
- Läuft im User-Kontext (kein sudo erforderlich)

**Service-Verwaltung:**

```bash
# Status prüfen
systemctl --user status cortex-server

# Logs anzeigen
journalctl --user -u cortex-server -f

# Service stoppen/starten/neu starten
systemctl --user stop cortex-server
systemctl --user start cortex-server
systemctl --user restart cortex-server

# Service deaktivieren (startet nicht mehr beim Login)
systemctl --user disable cortex-server
```

**Service-File anpassen:**

Das Service-File liegt in `~/.config/systemd/user/cortex-server.service`. Umgebungsvariablen können dort angepasst werden. Nach Änderungen:

```bash
systemctl --user daemon-reload
systemctl --user restart cortex-server
```

---

## Konfiguration

### Server (Umgebungsvariablen)

```bash
export CORTEX_DB_PATH="$HOME/.openclaw/cortex.db"
export CORTEX_PORT=9123
export CORTEX_LOG_LEVEL=info
export CORTEX_RATE_LIMIT=100
export CORTEX_RATE_LIMIT_WINDOW=1m
```

**Wichtig:** Für lokale Installationen ist **kein API-Key erforderlich**. API-Keys sind nur für Produktions-/Multi-User-Setups relevant.

### OpenClaw / Script-Client

In `.env` oder Umgebung (Cortex-Projekt: `cp .env.example .env` und anpassen):

```bash
CORTEX_API_URL=http://localhost:9123
CORTEX_APP_ID=openclaw
CORTEX_USER_ID=default
# API-Key nicht benötigt für lokale Installation
# Nur für Produktions-/Multi-User-Setups relevant:
# CORTEX_API_KEY=dein_geheimer_key
```

### Config-Datei (optional)

`~/.openclaw/cortex.json`:

```json
{
  "db_path": "~/.openclaw/cortex.db",
  "port": 9123,
  "log_level": "info",
  "rate_limit": 100,
  "rate_limit_window": "1m"
}
```

---

## Server starten & Health Check

### Server starten

```bash
make run
# oder
./cortex-server
# oder
go run ./cmd/cortex-server
```

### Health Check

```bash
curl http://localhost:9123/health
# oder: ./cortex-cli health
```

Erwartete Ausgabe: `{"status":"ok"}`

---

## Verwendung

### CLI (cortex-cli) – Empfohlen

Nach `make build` – empfohlene Befehle für alle Operationen:

#### Basis-Operationen

```bash
# Health Check
./cortex-cli health

# Memory speichern
./cortex-cli store "Der Nutzer mag Kaffee"
./cortex-cli store "User coffee preference" '{"type":"preference"}'

# Semantische Suche
./cortex-cli query "Kaffee" 10
./cortex-cli query "coffee preferences" 10 0.5
./cortex-cli query "coffee" 10 0.5 "1,2,3"  # Mit seedIds-Filter

# Memory löschen
./cortex-cli delete 1

# Statistiken
./cortex-cli stats
```

**Semantische Suche (query):** `query <text> [limit] [threshold] [seedIds]` – limit Standard 5, threshold Standard 0.2.

#### Agent Contexts (Session Persistence)

```bash
# Context erstellen
./cortex-cli context-create "my-agent" episodic '{"conversation":[],"lastTopic":"coffee"}'

# Contexts auflisten
./cortex-cli context-list "my-agent"

# Context abrufen
./cortex-cli context-get <id>
```

Memory-Typen: `episodic`, `semantic`, `procedural`, `working`.

#### Embeddings nachziehen (Batch)

```bash
./cortex-cli generate-embeddings 100
```

### API (curl)

#### Memory speichern

```bash
curl -X POST http://localhost:9123/seeds?appId=openclaw&externalUserId=user123 \
  -H "Content-Type: application/json" \
  -d '{"content": "Der Nutzer mag Kaffee", "metadata": {"type": "preference"}}'
```

#### Memory abfragen

```bash
curl -X POST http://localhost:9123/seeds/query?appId=openclaw&externalUserId=user123 \
  -H "Content-Type: application/json" \
  -d '{"query": "Kaffee", "limit": 5}'
```

#### Memory löschen

```bash
curl -X DELETE "http://localhost:9123/seeds/1?appId=openclaw&externalUserId=user123"
```

### Auto-Recall / Auto-Capture

Vor jeder AI-Interaktion Recall, nach jedem Austausch Capture (z. B. für OpenClaw):

**Recall: Relevante Memories abrufen**

```bash
# Mit cortex-cli direkt
./cortex-cli query "letzte User-Nachricht oder Thema" 10
```

**Capture: Neue Information speichern**

```bash
# Mit cortex-cli direkt
./cortex-cli store "Zusammenfassung oder Rohinhalt des Austauschs"
```

**Umgebungsvariablen für automatische Ausführung:**

- **CORTEX_AUTO_RECALL** (default: `true`): Bei `false` oder `0` sollte Recall übersprungen werden.
- **CORTEX_AUTO_CAPTURE** (default: `true`): Bei `false` oder `0` sollte Capture übersprungen werden.

### Typische Workflows

#### Workflow 1: Memory speichern und später abrufen

```bash
# 1. Memory speichern
./cortex-cli store "Der Nutzer bevorzugt Espresso am Morgen"

# 2. Später suchen
./cortex-cli query "Kaffee" 10

# 3. Spezifisches Memory löschen (falls nötig)
./cortex-cli delete <id>
```

#### Workflow 2: Agent Context für Session-Management

```bash
# 1. Context zu Beginn einer Session erstellen
./cortex-cli context-create "chatbot-session" episodic '{"conversation":[],"lastTopic":""}'

# 2. Während der Session: Context abrufen und aktualisieren
./cortex-cli context-get <id>

# 3. Alle Contexts eines Agents auflisten
./cortex-cli context-list "chatbot-session"
```

#### Workflow 3: Embeddings für bestehende Memories generieren

```bash
# 1. Prüfen, wie viele Memories noch keine Embeddings haben
./cortex-cli stats

# 2. Embeddings in Batches generieren (max 100 pro Batch)
./cortex-cli generate-embeddings 100

# 3. Erneut prüfen
./cortex-cli stats
```

---

## API-Endpunkte (Referenz)

| Method | Endpoint            | Beschreibung          |
|--------|----------------------|------------------------|
| GET    | /health              | Health Check           |
| POST   | /seeds               | Memory speichern       |
| POST   | /seeds/query         | Semantische Suche      |
| DELETE | /seeds/:id           | Memory löschen         |
| POST   | /agent-contexts      | Agent Context anlegen  |
| GET    | /agent-contexts      | Contexts auflisten     |
| GET    | /agent-contexts/{id} | Ein Context abrufen    |

Weitere: Bundles, Webhooks, Export/Import, Backup/Restore, Analytics (siehe Haupt-README).

---

## Neutron-Kompatibilität

Cortex ist als **lokale, Neutron-kompatible Alternative** gebaut. Gleiche Konzepte und API-Formate wie die [Neutron Memory API](https://openclaw.vanarchain.com/) / [OpenClaw Integration Guide](https://openclaw.vanarchain.com/guide-openclaw), aber Self-hosted ohne API-Key.

### API & Konzepte

| Neutron | Cortex | Kompatibel |
|--------|--------|------------|
| POST /seeds, POST /seeds/query, DELETE /seeds/:id | Identische Endpunkte | ✅ |
| Query: `?appId=...&externalUserId=...` oder im Body | Beides unterstützt | ✅ |
| Multi-Tenant (appId + externalUserId) | Identisch | ✅ |
| Semantische Suche, Similarity 0–1 | Lokale Embeddings (384-dim), Cosine-Similarity | ✅ |
| Bundles, Metadata (JSON) | Identisch | ✅ |
| Agent Contexts (episodic/semantic/procedural/working) | POST/GET /agent-contexts | ✅ |
| Bearer Token / API-Key | Nicht benötigt für lokale Installation; optional für Produktion: `CORTEX_API_KEY` + Header `X-API-Key` | ✅ (Auth optional) |

### Befehle

| Neutron-Guide | Cortex (cortex-cli) |
|---------------|---------------------|
| `test` | `cortex-cli health` |
| `save "content" "metadata"` | `cortex-cli store` |
| `search "query" [limit] [threshold] [seedIds]` | `cortex-cli query` (threshold, seedIds optional) |
| `context-create`, `context-list`, `context-get` | `cortex-cli context-create`, `context-list`, `context-get` |
| Auto-Recall / Auto-Capture | `cortex-cli query` / `cortex-cli store`; Env `CORTEX_AUTO_RECALL`, `CORTEX_AUTO_CAPTURE` |

### Umgebung (Env)

| Neutron | Cortex |
|---------|--------|
| API-Key, Agent-ID, User-ID | `CORTEX_API_URL`, `CORTEX_APP_ID`, `CORTEX_USER_ID`; `CORTEX_API_KEY` nur für Produktion (lokale Installation benötigt keinen API-Key) |

### Unterschiede

- **Deployment:** Neutron = Cloud (SaaS), Cortex = lokal (Self-hosted).
- **Datenbank:** Neutron = PostgreSQL + pgvector, Cortex = SQLite (pure-Go).
- **Embeddings:** Neutron = Jina v4 (Cloud), Cortex = lokaler Service (384-dim, offline).
- **Kosten:** Neutron = Pay-per-use, Cortex = kostenlos.

Ausführlich: [docs/CORTEX_NEUTRON_ALTERNATIVE.md](../docs/CORTEX_NEUTRON_ALTERNATIVE.md), [docs/VERGLEICH_NEUTRON.md](../docs/VERGLEICH_NEUTRON.md).

---

## Makefile-Targets

```bash
make help          # Alle Targets
make build         # cortex-server + cortex-cli
make run           # Server starten
make test          # Tests
make docker-build  # Docker Image
make docker-run    # Container starten
make install       # Beide Binaries nach /usr/local/bin
```

---

## Troubleshooting

### Port bereits belegt

```bash
export CORTEX_PORT=9124
go run ./cmd/cortex-server
```

### Datenbank-Fehler

```bash
ls -la ~/.openclaw/cortex.db
# Neu erstellen (ACHTUNG: Datenverlust!): rm ~/.openclaw/cortex.db
```

### Embeddings / Logs

```bash
CORTEX_LOG_LEVEL=debug go run ./cmd/cortex-server
```

### Server startet nicht

1. Prüfe, ob Go installiert ist: `go version`
2. Prüfe, ob Port frei ist: `lsof -ti:9123` (oder anderen Port)
3. Prüfe Logs: `CORTEX_LOG_LEVEL=debug make run`

### CLI-Befehle funktionieren nicht

1. Prüfe, ob Binary existiert: `ls -la cortex-cli`
2. Prüfe, ob Server läuft: `./cortex-cli health`
3. Prüfe Umgebungsvariablen: `echo $CORTEX_API_URL`

---

## Output Formats

CLI-Befehle geben strukturierte JSON-Ausgaben zurück. Für bessere Lesbarkeit:

```bash
# JSON-Output formatieren (mit jq)
./cortex-cli query "Kaffee" 10 | jq

# Oder direkt JSON anzeigen
./cortex-cli stats | jq '.total_memories'
```

---

## Tips

- Verwende `./cortex-cli help` für detaillierte Hilfe zu jedem Befehl
- Memory-IDs sind numerisch und inkrementell
- Semantische Suche funktioniert am besten mit vollständigen Sätzen oder Schlüsselwörtern
- Threshold-Werte zwischen 0.2-0.5 geben meist gute Ergebnisse
- Für Produktions-Setups: API-Key setzen und Rate-Limiting konfigurieren
- Embeddings werden automatisch beim Speichern generiert, können aber auch nachträglich mit `generate-embeddings` erstellt werden

---

## Dokumentation

- **README.md** – Projekt-Überblick, Build, Docker
- **docs/API.md** – Vollständige API
- **docs/CORTEX_NEUTRON_ALTERNATIVE.md** – Neutron-Alternative, OpenClaw-Guide-Vergleich
- **docs/VERGLEICH_NEUTRON.md** – Feature-für-Feature-Vergleich mit Neutron
- **scripts/README.md** – CLI-Tool Dokumentation

## Lizenz

Siehe `LICENSE` im Hauptverzeichnis.
