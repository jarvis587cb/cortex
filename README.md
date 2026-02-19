# Cortex – Gehirn-Backend für OpenClaw

Cortex ist ein **leichtgewichtiges Go-Backend** mit SQLite-Datenbank, das als persistentes „Gehirn“ für OpenClaw-Agenten dient. Es speichert Erinnerungen (Memories), Entities mit Fakten sowie Relationen zwischen Entities und wird über ein OpenClaw-Plugin angebunden.

## Features

- ✅ **Persistente Speicherung**: Erinnerungen, Fakten und Relationen in SQLite
- ✅ **Semantische Suche mit Embeddings**: Vektor-basierte Suche für bessere Ergebnisse
- ✅ **Multimodal-Support**: Unterstützung für Text, Bilder und Dokumente
- ✅ **Jina API Integration**: Nutzung von Jina v4 Embeddings (wie Neutron)
- ✅ **Fallback-Lösung**: Lokaler Embedding-Service wenn keine API verfügbar
- ✅ **Multi-Tenant-Support**: Isolation durch `appId` + `externalUserId`
- ✅ **Neutron-kompatibel**: Gleiche API-Formate wie neutron-local
- ✅ **Leichtgewichtig**: Pure-Go (kein cgo), keine externen Dependencies außer SQLite
- ✅ **REST-API**: Einfache HTTP-Endpunkte für alle Operationen
- ✅ **OpenClaw-Integration**: TypeScript-Plugin mit Agent-Tools

## Features

- ✅ **Semantische Suche mit Embeddings**: Vektor-basierte Suche für bessere Ergebnisse
- ✅ **Multimodal-Support**: Unterstützung für Text, Bilder und Dokumente
- ✅ **Jina API Integration**: Nutzung von Jina v4 Embeddings (wie Neutron)
- ✅ **Fallback-Lösung**: Lokaler Embedding-Service wenn keine API verfügbar
- ✅ **Neutron-kompatible API**: Seeds-API für einfache Integration
- ✅ **Multi-Tenant**: Isolation von Daten nach `appId` und `externalUserId`
- ✅ **REST API**: Vollständige HTTP-API für alle Operationen
- ✅ **SQLite**: Leichtgewichtige, embedded Datenbank
- ✅ **Docker Support**: Containerisierung für einfache Deployment

## Architektur

Cortex besteht aus folgenden Komponenten:

### 1. Go-Server (`cortex`)

**Backend-Service** mit SQLite-Datenbank und HTTP-API:

- **Datenbank**: SQLite (`~/.openclaw/cortex.db` oder über `CORTEX_DB_PATH`)
- **Port**: 9123 (Standard) oder über `CORTEX_PORT`
- **Technologie**: Go 1.23+, GORM, `github.com/glebarez/sqlite` (pure-Go)
- **Code-Struktur**: 
  - `cmd/cortex/main.go` – Server-Start und Routing
  - `internal/models/` – Datenmodelle
  - `internal/store/` – Datenbank-Operationen
  - `internal/api/` – HTTP-Handler
  - `internal/helpers/` – Utility-Funktionen
  - `internal/middleware/` – HTTP-Middleware
  - `internal/embeddings/` – Embedding-Generierung und semantische Suche

### 2. Scripts (`scripts/`)

**Bash-Scripts** für CLI, Tests und Benchmarks:

- `cortex-cli.sh` – CLI-Tool für alle API-Operationen
- `benchmark.sh` – Performance-Benchmarks
- `test-e2e.sh` – End-to-End-Tests
- `lib/common.sh` – Gemeinsame Funktionen für Scripts

Siehe [scripts/README.md](scripts/README.md) für Details.

### 3. OpenClaw-Plugin (geplant)

**TypeScript-Plugin** für OpenClaw-Agenten (in Entwicklung):

- Registriert Agent-Tools für Memory-Operationen
- Ruft die Go-API über HTTP auf
- Unterstützt Multi-Tenant-Konfiguration

## Installation & Start

### Go-Server starten

```bash
cd projects/cortex
go mod tidy
go run ./...
```

**Umgebungsvariablen** (optional):

- `CORTEX_DB_PATH` – Pfad zur SQLite-Datei (Standard: `~/.openclaw/cortex.db`)
- `CORTEX_PORT` – Port (Standard: `9123`)
- `CORTEX_API_KEY` – API-Key für Authentifizierung (optional, deaktiviert Auth wenn nicht gesetzt)
- `CORTEX_LOG_LEVEL` – Log-Level (debug, info, warn, error, Standard: info)
- `JINA_API_KEY` – API-Key für Jina Embeddings (optional, für semantische Suche)
- `JINA_API_URL` – URL für Jina API (Standard: `https://api.jina.ai/v1/embeddings`)

**Health-Check:**

```bash
curl http://localhost:9123/health
# {"status":"ok","timestamp":"2026-02-19T15:00:00Z"}
```

### CLI-Tool verwenden

Das `cortex-cli.sh` Script bietet eine einfache CLI für alle API-Operationen:

```bash
# Health Check
./scripts/cortex-cli.sh health

# Memory speichern
./scripts/cortex-cli.sh store "Der Nutzer mag Kaffee"

# Memory-Suche
./scripts/cortex-cli.sh query "Kaffee" 10

# Memory löschen
./scripts/cortex-cli.sh delete 1

# Statistiken
./scripts/cortex-cli.sh stats
```

**Umgebungsvariablen für CLI:**
- `CORTEX_API_URL` – API Base URL (Standard: `http://localhost:9123`)
- `CORTEX_APP_ID` – App-ID für Multi-Tenant (Standard: `openclaw`)
- `CORTEX_USER_ID` – User-ID für Multi-Tenant (Standard: `default`)

Siehe [scripts/README.md](scripts/README.md) für weitere Details.

### Tests ausführen

```bash
# End-to-End-Tests
./scripts/test-e2e.sh

# Performance-Benchmark
./scripts/benchmark.sh 50
```

### OpenClaw-Plugin (geplant)

Das TypeScript-Plugin für OpenClaw-Agenten ist in Entwicklung. Nach Installation:

1. **Config in `~/.openclaw/openclaw.json`:**

   ```json5
   {
     plugins: {
       entries: {
         cortex: {
           enabled: true,
           config: {
             url: "http://localhost:9123",
             appId: "openclaw",        // optional, Standard: "openclaw"
             externalUserId: "default"  // optional, Standard: "default"
           }
         }
       }
     }
   }
   ```

2. **Tools im Agent aktivieren:**

   ```json5
   {
     agents: {
       list: [
         {
           id: "main",
           tools: {
             allow: [
               "cortex",          // alle Cortex-Tools
               "store_memory",    // oder einzelne Tools
               "query_memory",
               "delete_memory"
             ]
           }
         }
       ]
     }
   }
   ```

## Embeddings & Semantische Suche

Cortex unterstützt semantische Suche mit Embeddings für bessere Suchergebnisse:

### Konfiguration

**Mit Jina API (empfohlen):**
```bash
export JINA_API_KEY="dein-jina-api-key"
export JINA_API_URL="https://api.jina.ai/v1/embeddings"  # Optional
```

**Ohne API (Fallback):**
Cortex verwendet automatisch einen lokalen Embedding-Service als Fallback.

### Automatische Embedding-Generierung

Beim Speichern von Memories werden automatisch Embeddings generiert (asynchron):

```bash
# Memory speichern - Embedding wird automatisch generiert
curl -X POST http://localhost:9123/seeds \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "appId": "myapp",
    "externalUserId": "user123",
    "content": "Der Benutzer mag Kaffee und liest gerne Bücher",
    "metadata": {"source": "chat"}
  }'
```

### Batch-Embedding-Generierung

Für bestehende Memories ohne Embeddings:

```bash
# Generiere Embeddings für bis zu 10 Memories
curl -X POST "http://localhost:9123/seeds/generate-embeddings?batchSize=10" \
  -H "X-API-Key: dein-key"
```

### Semantische Suche

Die Query-API nutzt automatisch semantische Suche wenn Embeddings verfügbar sind:

```bash
curl -X POST http://localhost:9123/seeds/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "appId": "myapp",
    "externalUserId": "user123",
    "query": "Was mag der Benutzer trinken?",
    "limit": 5
  }'
```

Die Antwort enthält `similarity`-Scores (0.0-1.0) basierend auf Cosine-Similarity.

### Multimodal-Support

Cortex erkennt automatisch verschiedene Content-Types:

- **Text**: Standard-Text-Embeddings
- **Bilder**: Multimodal-Embeddings (wenn Jina API verfügbar)
- **Dokumente**: PDF- und Dokument-Embeddings

Content-Type wird automatisch erkannt aus:
- Metadata (`contentType` oder `content_type`)
- Base64-encoded Bilder (`data:image/...`)
- URLs mit Dateiendungen (`.jpg`, `.png`, `.pdf`)

## API-Endpunkte

### Neutron-kompatible Seeds-API

Kompatibel mit neutron-local (gleiche Request/Response-Formate):

#### `POST /seeds` – Memory speichern

```bash
curl -X POST http://localhost:9123/seeds \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "openclaw",
    "externalUserId": "user1",
    "content": "Der Nutzer mag Kaffee mit Hafermilch",
    "metadata": {"tags": ["preferences", "coffee"]}
  }'
```

**Response:**
```json
{
  "id": 1,
  "message": "Memory stored successfully"
}
```

#### `POST /seeds/query` – Memory-Suche

```bash
curl -X POST http://localhost:9123/seeds/query \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "openclaw",
    "externalUserId": "user1",
    "query": "Kaffee-Präferenzen",
    "limit": 5
  }'
```

**Response:**
```json
[
  {
    "id": 1,
    "content": "Der Nutzer mag Kaffee mit Hafermilch",
    "metadata": {"tags": ["preferences", "coffee"]},
    "created_at": "2026-02-19T15:00:00Z",
    "similarity": 0.95
  }
]
```

**Hinweis:** `similarity` wird basierend auf Cosine-Similarity der Embeddings berechnet (0.0-1.0). Wenn keine Embeddings verfügbar sind, wird eine Text-basierte Heuristik verwendet.

#### `POST /seeds/generate-embeddings` – Embeddings generieren

```bash
curl -X POST "http://localhost:9123/seeds/generate-embeddings?batchSize=10" \
  -H "X-API-Key: dein-key"
```

Generiert Embeddings für bestehende Memories ohne Embedding. `batchSize` bestimmt, wie viele Memories pro Aufruf verarbeitet werden (Standard: 10, Max: 100).

**Response:**
```json
{
  "message": "Embeddings generation started"
}
```

#### `DELETE /seeds/:id` – Memory löschen

```bash
curl -X DELETE "http://localhost:9123/seeds/1?appId=openclaw&externalUserId=user1"
```

**Response:**
```json
{
  "message": "Memory deleted successfully",
  "id": 1
}
```

### Cortex-API (Original)

Zusätzliche Endpunkte für erweiterte Features:

#### `POST /remember` – Erinnerung speichern

```bash
curl -X POST http://localhost:9123/remember \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Der Nutzer mag Kaffee mit Hafermilch",
    "type": "semantic",
    "entity": "user:jarvis",
    "tags": "preference,coffee",
    "importance": 7
  }'
```

#### `GET /recall` – Erinnerungen abrufen

```bash
curl "http://localhost:9123/recall?q=Kaffee&limit=5"
```

#### `POST /entities?entity=...` – Fakt setzen

```bash
curl -X POST "http://localhost:9123/entities?entity=user:jarvis" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "favorite_coffee",
    "value": "Latte mit Hafermilch"
  }'
```

#### `GET /entities?name=...` – Entity abrufen

```bash
curl "http://localhost:9123/entities?name=user:jarvis"
```

#### `POST /relations` – Relation hinzufügen

```bash
curl -X POST http://localhost:9123/relations \
  -H "Content-Type: application/json" \
  -d '{
    "from": "user:jarvis",
    "to": "user:alice",
    "type": "friend"
  }'
```

#### `GET /stats` – Statistiken

```bash
curl http://localhost:9123/stats
```

**Response:**
```json
{
  "memories": 42,
  "entities": 5,
  "relations": 12
}
```

## Agent-Tools (geplant)

Das zukünftige Plugin wird folgende Tools für OpenClaw-Agenten registrieren:

### Neutron-kompatible Tools

- **`store_memory`** – Memory speichern (Multi-Tenant)
- **`query_memory`** – Memory-Suche durchführen
- **`delete_memory`** – Memory löschen (tenant-sicher)
- **`health_check`** – API-Status prüfen

### Cortex-Tools

- **`cortex_remember`** – Erinnerung speichern
- **`cortex_recall`** – Erinnerungen abrufen
- **`cortex_fact_set`** – Fakt für Entity setzen
- **`cortex_fact_get`** – Fakten für Entity abrufen
- **`cortex_relation_add`** – Relation hinzufügen
- **`cortex_stats`** – Statistiken abrufen

**Hinweis:** Bis das Plugin verfügbar ist, können alle Operationen über die REST-API oder das CLI-Tool (`scripts/cortex-cli.sh`) verwendet werden.

## Datenmodell

### Memories

- `id` – Eindeutige ID
- `content` – Textinhalt
- `type` – Typ (z.B. "semantic", "episodic")
- `entity` – Optionale Entity-Zuordnung
- `tags` – Kommagetrennte Tags
- `importance` – Wichtigkeit (1-10)
- `app_id` – Multi-Tenant: App-ID
- `external_user_id` – Multi-Tenant: User-ID
- `metadata` – JSON-Metadaten (als Text)
- `created_at` – Zeitstempel

### Entities

- `id` – Eindeutige ID
- `name` – Entity-Name (unique)
- `data` – JSON-Objekt mit Fakten (als Text)
- `created_at`, `updated_at` – Zeitstempel

### Relations

- `id` – Eindeutige ID
- `from_entity` – Quell-Entity
- `to_entity` – Ziel-Entity
- `type` – Relationstyp (z.B. "friend", "owns")
- `valid_from`, `valid_to` – Optionale Gültigkeitszeiträume
- `created_at` – Zeitstempel

## Neutron-Kompatibilität

Cortex bietet eine **neutron-kompatible Seeds-API** ohne Embeddings:

- ✅ Gleiche Endpunkte (`/seeds`, `/seeds/query`, `/seeds/:id`)
- ✅ Gleiche Request/Response-Formate
- ✅ Multi-Tenant-Support (`appId`, `externalUserId`)
- ⚠️ **Textsuche statt semantischer Suche** (kein pgvector/Transformers.js nötig)

**Unterschiede zu neutron-local:**

- Keine Embeddings: Textsuche mit `LIKE` statt Cosine-Similarity
- Kein PostgreSQL: SQLite statt pgvector
- Kein Transformers.js: Reines Go-Backend
- Gleiche API-Formate: Kompatibel mit neutron-Skills/Tools

Die bestehende Cortex-API (`/remember`, `/recall`, etc.) bleibt für Rückwärtskompatibilität erhalten.

## Entwicklung

### Dependencies

**Go:**
- `github.com/glebarez/sqlite` – Pure-Go SQLite-Treiber
- `gorm.io/gorm` – ORM

**TypeScript (Plugin):**
- `@sinclair/typebox` – Schema-Validierung
- `@types/node` – Node.js-Typen

### Tests

Das Projekt enthält umfassende Unit-Tests:

```bash
# Alle Tests ausführen
go test ./...

# Mit Verbose-Output
go test -v ./...

# Mit Coverage-Report
go test -cover ./...
```

### Authentifizierung

Cortex unterstützt optionale API-Key-Authentifizierung:

- **Ohne API-Key:** Alle Endpunkte sind öffentlich (Development-Modus)
- **Mit API-Key:** Alle Endpunkte außer `/health` erfordern Authentifizierung

**Verwendung:**

```bash
# Server mit API-Key starten
CORTEX_API_KEY=your-secret-key go run ./...

# API-Requests mit API-Key
curl -H "Authorization: Bearer your-secret-key" \
  http://localhost:9123/seeds \
  -X POST -H "Content-Type: application/json" \
  -d '{"appId":"test","externalUserId":"user1","content":"Test"}'
```

### Logging

Cortex verwendet strukturiertes Logging (log/slog):

- **Log-Level:** Über `CORTEX_LOG_LEVEL` konfigurierbar (debug, info, warn, error)
- **Strukturiert:** Alle Logs enthalten strukturierte Felder für besseres Parsing
- **Format:** Text-Format (kann zu JSON geändert werden)

**Beispiel-Logs:**
```
level=INFO msg="cortex server starting" addr=:9123 db=/path/to/cortex.db
level=DEBUG msg="authenticated request" path=/seeds method=POST
level=ERROR msg="remember insert error" error="..."
```

### Build

```bash
# Go-Binary bauen
go build -o cortex ./cmd/cortex

# Oder direkt ausführen
go run ./cmd/cortex

# Tests ausführen
go test ./...

# Tests mit Coverage
go test -cover ./...
```

### Docker

```bash
# Docker Image bauen
docker build -t cortex .

# Mit docker-compose starten
docker-compose up -d

# Oder direkt mit Docker
docker run -d \
  -p 9123:9123 \
  -e CORTEX_API_KEY=your-secret-key \
  -v cortex-data:/data \
  cortex
```

### Scripts verwenden

Die Bash-Scripts benötigen:
- `curl` – HTTP-Requests
- `jq` – JSON-Verarbeitung (optional, aber empfohlen)

**Installation:**
```bash
# Ubuntu/Debian
sudo apt-get install curl jq

# macOS
brew install curl jq
```

## Troubleshooting

### Port bereits belegt

```bash
# Anderen Port verwenden
CORTEX_PORT=9124 go run ./...
```

### Datenbank-Fehler

```bash
# Datenbank-Pfad prüfen
ls -la ~/.openclaw/cortex.db

# Datenbank löschen (Vorsicht: Datenverlust!)
rm ~/.openclaw/cortex.db
```

### API nicht erreichbar

```bash
# Prüfe ob Server läuft
curl http://localhost:9123/health

# Prüfe Logs
# (Server-Logs werden auf stdout ausgegeben)
```

### Script-Fehler

```bash
# Prüfe Dependencies
command -v curl && command -v jq

# Prüfe API-URL
echo $CORTEX_API_URL
```

## Lizenz

MIT (oder wie im Workspace definiert)
