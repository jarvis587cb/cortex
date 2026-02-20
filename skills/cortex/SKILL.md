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

## Voraussetzungen

- **Go 1.23+** (`go version`)
- **Git** und **Make**

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
make build    # Erstellt cortex-server und cortex-cli
make run      # Startet den Server
```

### Binaries installieren

```bash
make install  # Installiert beide Binaries nach /usr/local/bin
```

### systemd User Service

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

**Hinweis:** Für lokale Installationen ist kein API-Key erforderlich.

### Client-Konfiguration

```bash
export CORTEX_API_URL=http://localhost:9123
export CORTEX_APP_ID=openclaw
export CORTEX_USER_ID=default
```

---

## Server starten

```bash
make run
# oder
./cortex-server
```

**Health Check:**
```bash
curl http://localhost:9123/health
# oder
./cortex-cli health
```

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

**Query-Syntax:** `query <text> [limit] [threshold] [seedIds]` – Standard: limit=5, threshold=0.2

#### Agent Contexts (Session Persistence)

```bash
# Context erstellen
./cortex-cli context-create "my-agent" episodic '{"conversation":[],"lastTopic":"coffee"}'

# Contexts auflisten
./cortex-cli context-list "my-agent"

# Context abrufen
./cortex-cli context-get <id>
```

**Memory-Typen:** `episodic`, `semantic`, `procedural`, `working`

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

**Umgebungsvariablen:**
- `CORTEX_AUTO_RECALL` (default: `true`) – Recall deaktivieren mit `false` oder `0`
- `CORTEX_AUTO_CAPTURE` (default: `true`) – Capture deaktivieren mit `false` oder `0`

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

Weitere Endpunkte: Bundles, Webhooks, Export/Import, Backup/Restore, Analytics (siehe `docs/API.md`).

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
make help    # Hilfe anzeigen
make build   # Beide Binaries bauen
make run     # Server starten
make test    # Tests ausführen
make install # Binaries nach /usr/local/bin installieren
make clean   # Build-Artefakte entfernen
```

---

## Troubleshooting

### Häufige Probleme

**Port bereits belegt:**
```bash
export CORTEX_PORT=9124 && make run
```

**Server startet nicht:**
- Go installiert? `go version`
- Port frei? `lsof -ti:9123`
- Logs prüfen: `CORTEX_LOG_LEVEL=debug make run`

**CLI-Befehle funktionieren nicht:**
- Binary vorhanden? `ls -la cortex-cli`
- Server läuft? `./cortex-cli health`
- Umgebungsvariablen? `echo $CORTEX_API_URL`

**Datenbank zurücksetzen:**
```bash
rm ~/.openclaw/cortex.db  # ACHTUNG: Datenverlust!
```

---

## Lizenz

Siehe `LICENSE` im Hauptverzeichnis.
