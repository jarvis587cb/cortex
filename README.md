# Cortex – Gehirn-Backend für OpenClaw

Cortex ist ein **leichtgewichtiges Go-Backend** mit SQLite-Datenbank, das als persistentes „Gehirn“ für OpenClaw-Agenten dient. Es speichert Erinnerungen (Memories), Entities mit Fakten sowie Relationen zwischen Entities und wird über ein OpenClaw-Plugin angebunden.

## Features

- ✅ **Persistente Speicherung**: Erinnerungen, Fakten und Relationen in SQLite
- ✅ **Semantische Suche mit Embeddings**: Vektor-basierte Suche für bessere Ergebnisse
- ✅ **Lokaler Embedding-Service**: Vollständig lokale Embedding-Generierung ohne externe APIs
- ✅ **Multi-Tenant-Support**: Isolation durch `appId` + `externalUserId`
- ✅ **Neutron-kompatibel**: Gleiche API-Formate wie neutron-local
- ✅ **Bundles**: Organisation von Memories in logische Gruppen
- ✅ **Webhooks**: Event-Benachrichtigungen für Memory-Änderungen
- ✅ **Analytics**: Dashboard-Daten über API
- ✅ **Export/Import**: Daten-Migration unterstützt
- ✅ **Backup/Restore**: Datenbank-Backup verfügbar
- ✅ **Rate Limiting**: Token-Bucket-Algorithmus für API-Schutz
- ✅ **Leichtgewichtig**: Pure-Go (kein cgo), keine externen Dependencies außer SQLite
- ✅ **REST-API**: Einfache HTTP-Endpunkte für alle Operationen
- ✅ **TypeScript SDK**: Vollständiges SDK für einfache Integration
- ✅ **Docker Support**: Containerisierung für einfaches Deployment

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
- `cortex-memory.sh` – Neutron-kompatibles Script (save, search, context-*), siehe [skills/cortex-memory/SKILL.md](skills/cortex-memory/SKILL.md)
- `api-key.sh` – API-Key anlegen/löschen (CORTEX_API_KEY in .env)
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

### Konfiguration (optional)

Die Datei `.env` wird nicht ins Repository committed (steht in `.gitignore`). Für lokale Anpassungen:

```bash
cp .env.example .env
# .env bearbeiten (z. B. CORTEX_PORT, CORTEX_API_KEY)
```

API-Keys anlegen/entfernen: `./scripts/api-key.sh create` bzw. `delete` (siehe [scripts/README.md](scripts/README.md)).

### Go-Server starten

```bash
# Ins Cortex-Projektverzeichnis wechseln
cd /path/to/cortex   # bzw. z. B. cd ~/.openclaw/workspace/projects/cortex
go mod tidy
go run ./...
```

**Umgebungsvariablen** (optional):

- `CORTEX_DB_PATH` – Pfad zur SQLite-Datei (Standard: `~/.openclaw/cortex.db`)
- `CORTEX_PORT` – Port (Standard: `9123`)
- `CORTEX_LOG_LEVEL` – Log-Level (debug, info, warn, error, Standard: info)
- `CORTEX_RATE_LIMIT` – Rate Limit (Requests pro Zeitfenster, Standard: 100, 0 = deaktiviert)
- `CORTEX_RATE_LIMIT_WINDOW` – Rate Limit Zeitfenster (Standard: `1m`)
- `CORTEX_API_KEY` – optional; wenn gesetzt, müssen Requests `Authorization: Bearer <key>` oder `X-API-Key: <key>` senden (außer `GET /health`)

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

Cortex unterstützt semantische Suche mit **vollständig lokalen Embeddings**.

### Embedding-Service-Auswahl

Cortex verwendet standardmäßig den lokalen Embedding-Service:

**Lokaler Embedding-Service:**
- ✅ **384-dimensionale Embeddings** - Lokale Hash-basierte Generierung
- ✅ **Vollständig offline** - Keine externe API nötig
- ✅ **Keine API-Keys** - Funktioniert ohne Konfiguration
- ✅ **Text-Support** - Optimiert für Text-Inhalte
- ✅ **Schnell** - Keine Netzwerk-Latenz
- ✅ **Hash-basierter Algorithmus** - Basierend auf Content-Analyse und Wort-Frequenzen
- ✅ **Synonym-Erweiterung** - Begriffe wie Kaffee/Latte/Espresso werden verknüpft für bessere begriffliche Treffer

### Automatische Embedding-Generierung

Beim Speichern von Memories werden automatisch Embeddings generiert (synchron, damit Suche sofort funktioniert):

```bash
# Memory speichern - Embedding wird automatisch generiert
curl -X POST http://localhost:9123/seeds \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "myapp",
    "externalUserId": "user123",
    "content": "Der Benutzer mag Kaffee und liest gerne Bücher",
    "metadata": {"source": "chat"}
  }'
```

### Batch-Embedding-Generierung

Für bestehende Memories ohne Embeddings oder nach Änderungen am Embedder (z. B. neue Synonyme):

```bash
# Generiere Embeddings für bis zu 10 Memories
curl -X POST "http://localhost:9123/seeds/generate-embeddings?batchSize=10" \
```

### Semantische Suche

Die Query-API nutzt automatisch semantische Suche wenn Embeddings verfügbar sind:

```bash
curl -X POST http://localhost:9123/seeds/query \
  -H "Content-Type: application/json" \
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
- **Bilder**: Content-Type-Erkennung für Bild-URLs und Base64-Daten
- **Dokumente**: PDF- und Dokument-URLs werden erkannt

Content-Type wird automatisch erkannt aus:
- Metadata (`contentType` oder `content_type`)
- Base64-encoded Bilder (`data:image/...`)
- URLs mit Dateiendungen (`.jpg`, `.png`, `.pdf`)

**Hinweis:** Der lokale Embedding-Service generiert für alle Content-Types semantische Vektoren basierend auf Text-Analyse. Für echte Bild-Embeddings wäre eine externe API oder ein lokales Modell erforderlich.

## Bundles

Cortex unterstützt **Bundles** zur Organisation von Memories in logische Gruppen:

### Bundle erstellen

```bash
curl -X POST "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -d '{"name": "Coffee Preferences"}'
```

### Bundles auflisten

```bash
curl "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
```

### Memory in Bundle speichern

```bash
curl -X POST "http://localhost:9123/seeds?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Lieblingskaffee: Latte mit Hafermilch",
    "bundleId": 1
  }'
```

### Memories in Bundle suchen

```bash
curl -X POST "http://localhost:9123/seeds/query?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Kaffee",
    "bundleId": 1,
    "limit": 10
  }'
```

## TypeScript SDK

Cortex bietet ein offizielles TypeScript SDK für einfache Integration:

### Installation

```bash
cd sdk
npm install
npm run build
```

### Verwendung

```typescript
import { CortexClient } from "@cortex/memory-sdk";

const client = new CortexClient({
  baseUrl: "http://localhost:9123",
  appId: "myapp",
  externalUserId: "user123",
});

// Memory speichern
const memory = await client.storeMemory({
  appId: "myapp",
  externalUserId: "user123",
  content: "Der Benutzer mag Kaffee",
  metadata: { source: "chat" },
});

// Memory-Suche
const results = await client.queryMemory({
  appId: "myapp",
  externalUserId: "user123",
  query: "Was mag der Benutzer?",
  limit: 5,
});

// Bundle erstellen
const bundle = await client.createBundle({
  appId: "myapp",
  externalUserId: "user123",
  name: "Coffee Preferences",
});
```

Siehe [sdk/README.md](sdk/README.md) für vollständige Dokumentation.

## API-Endpunkte

### Neutron-kompatible Seeds-API

Vollständig kompatibel mit Neutron Memory API (gleiche Request/Response-Formate):

**Unterstützt beide Parameter-Formate:**
- **Query-Parameter** (Neutron-Style): `?appId=xxx&externalUserId=yyy`
- **Body-Parameter** (Cortex-Style): `{ "appId": "xxx", "externalUserId": "yyy" }`

#### `POST /seeds` – Memory speichern

**Mit Query-Parameter (Neutron-Style):**
```bash
curl -X POST "http://localhost:9123/seeds?appId=openclaw&externalUserId=user1" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Der Nutzer mag Kaffee mit Hafermilch",
    "metadata": {"tags": ["preferences", "coffee"]},
    "bundleId": 1
  }'
```

**Mit Body-Parameter (Cortex-Style):**
```bash
curl -X POST http://localhost:9123/seeds \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "openclaw",
    "externalUserId": "user1",
    "content": "Der Nutzer mag Kaffee mit Hafermilch",
    "metadata": {"tags": ["preferences", "coffee"]},
    "bundleId": 1
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

**Mit Query-Parameter (Neutron-Style):**
```bash
curl -X POST "http://localhost:9123/seeds/query?appId=openclaw&externalUserId=user1" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Kaffee-Präferenzen",
    "limit": 5,
    "bundleId": 1
  }'
```

**Mit Body-Parameter (Cortex-Style):**
```bash
curl -X POST http://localhost:9123/seeds/query \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "openclaw",
    "externalUserId": "user1",
    "query": "Kaffee-Präferenzen",
    "limit": 5,
    "bundleId": 1
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

## Bundles API

### `POST /bundles` – Bundle erstellen

```bash
curl -X POST "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -d '{"name": "Coffee Preferences"}'
```

### `GET /bundles` – Bundles auflisten

```bash
curl "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
```

### `GET /bundles/:id` – Bundle abrufen

```bash
curl "http://localhost:9123/bundles/1?appId=myapp&externalUserId=user123" \
```

### `DELETE /bundles/:id` – Bundle löschen

```bash
curl -X DELETE "http://localhost:9123/bundles/1?appId=myapp&externalUserId=user123" \
```

**Hinweis:** Beim Löschen eines Bundles bleiben die Memories erhalten, `bundleId` wird auf `NULL` gesetzt.

## Export/Import

Cortex unterstützt **Export und Import** von Daten:

### Daten exportieren

```bash
curl "http://localhost:9123/export?appId=myapp&externalUserId=user123" \
  -o cortex-export.json
```

### Daten importieren

```bash
curl -X POST "http://localhost:9123/import?appId=myapp&externalUserId=user123&overwrite=false" \
  -H "Content-Type: application/json" \
  -d @cortex-export.json
```

## Backup/Restore

Cortex unterstützt **Backup und Restore** der Datenbank:

### Backup erstellen

```bash
curl -X POST "http://localhost:9123/backup?path=/backups/cortex-backup.db" \
```

### Restore durchführen

```bash
curl -X POST "http://localhost:9123/restore?path=/backups/cortex-backup.db" \
```

**⚠️ WICHTIG:** Nach dem Restore muss der Server neu gestartet werden!

## Analytics

Cortex bietet **Analytics-Endpunkte** für Dashboard-Daten:

### Analytics abrufen

```bash
# Tenant-spezifische Analytics
curl "http://localhost:9123/analytics?appId=myapp&externalUserId=user123&days=30" \

# Globale Analytics
curl "http://localhost:9123/analytics?days=30" \
```

**Verfügbare Metriken:**
- Gesamtanzahl Memories, Bundles, Webhooks
- Memories mit Embeddings
- Aufschlüsselung nach Type und Bundle
- Recent Activity (letzte 50 Aktivitäten)
- Storage-Statistiken

## Agent-Tools (geplant)

Das zukünftige Plugin wird folgende Tools für OpenClaw-Agenten registrieren:

### Neutron-kompatible Tools

- **`store_memory`** – Memory speichern (Multi-Tenant)
- **`query_memory`** – Memory-Suche durchführen
- **`delete_memory`** – Memory löschen (tenant-sicher)
- **`create_bundle`** – Bundle erstellen
- **`list_bundles`** – Bundles auflisten
- **`delete_bundle`** – Bundle löschen
- **`health_check`** – API-Status prüfen

### Cortex-Tools

- **`cortex_remember`** – Erinnerung speichern
- **`cortex_recall`** – Erinnerungen abrufen
- **`cortex_fact_set`** – Fakt für Entity setzen
- **`cortex_fact_get`** – Fakten für Entity abrufen
- **`cortex_relation_add`** – Relation hinzufügen
- **`cortex_stats`** – Statistiken abrufen

**Hinweis:** Bis das Plugin verfügbar ist, können alle Operationen über die REST-API, das TypeScript SDK oder das CLI-Tool (`scripts/cortex-cli.sh`) verwendet werden.

## Datenmodell

### Memories (Seeds)

- `id` – Eindeutige ID
- `content` – Textinhalt
- `type` – Typ (z. B. "semantic", "episodic")
- `entity` – Optionale Entity-Zuordnung
- `tags` – Kommagetrennte Tags
- `importance` – Wichtigkeit (1–10)
- `app_id` – Multi-Tenant: App-ID
- `external_user_id` – Multi-Tenant: User-ID
- `bundle_id` – Optionale Bundle-Zuordnung
- `metadata` – JSON-Metadaten (als Text)
- `content_type` – Content-Type (z. B. "text/plain")
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

## Cortex als Neutron-Alternative

Cortex ist eine **vollständig lokale, kostenlose Alternative** zur Neutron Memory API von Vanar. Während Neutron eine Cloud-basierte SaaS-Lösung ist, bietet Cortex dieselben Features als Self-hosted Lösung ohne externe Abhängigkeiten.

### Kern-Features (Neutron-kompatibel)

- ✅ **Persistent Semantic Memory**: Cross-Session Context, Memory überlebt Neustarts
- ✅ **Seeds API**: Identische Endpunkte (`/seeds`, `/seeds/query`, `/seeds/:id`)
- ✅ **Semantic Search**: Vector-Embeddings mit Cosine-Similarity (<200ms für typische Use-Cases)
- ✅ **Multi-Tenant Support**: Sichere Isolation durch `appId` + `externalUserId`
- ✅ **REST API + TypeScript SDK**: Production-ready, vollständig kompatibel
- ✅ **Bundles**: Organisation von Memories in logische Gruppen
- ✅ **Cross-Platform Continuity**: Gemeinsames Memory über Discord/Slack/WhatsApp/Web

### Vorteile von Cortex

- 🏠 **Lokal**: Keine Cloud-Abhängigkeit, vollständig Self-hosted
- 💰 **Kostenlos**: Keine laufenden API-Kosten
- 🔒 **Privacy**: 100% lokale Datenhaltung
- ⚙️ **Kontrolle**: Volle Kontrolle über Infrastruktur und Daten
- 🚀 **Schnell**: Keine Netzwerk-Latenz, lokale Performance

### Dokumentation

- **[docs/CORTEX_NEUTRON_ALTERNATIVE.md](docs/CORTEX_NEUTRON_ALTERNATIVE.md)** – Feature-für-Feature Vergleich mit Neutron-Artikel-Anforderungen
- **[docs/INTEGRATION_GUIDE.md](docs/INTEGRATION_GUIDE.md)** – Cross-Platform Integration Guide (Discord/Slack/WhatsApp/Web)
- **[docs/PERFORMANCE.md](docs/PERFORMANCE.md)** – Performance-Benchmarks und Optimierungen
- **[docs/CRYPTO_EVALUATION.md](docs/CRYPTO_EVALUATION.md)** – Evaluierung kryptographischer Verifizierung
- **[docs/VERGLEICH_NEUTRON.md](docs/VERGLEICH_NEUTRON.md)** – Detaillierter Feature-Vergleich mit Neutron

### Migration von Neutron

**Minimale Code-Änderungen:**

```typescript
// Vorher (Neutron)
import { NeutronClient } from '@vanar/neutron-sdk';
const client = new NeutronClient({
    apiKey: 'nk_...',
    baseUrl: 'https://api-neutron.vanarchain.com'
});

// Nachher (Cortex) – nur Base-URL ändern, kein API-Key nötig
import { CortexClient } from '@cortex/memory-sdk';
const client = new CortexClient({
    baseUrl: 'http://localhost:9123' // Lokaler Server
});

// API-Calls bleiben identisch
await client.storeMemory({...});
await client.queryMemory({...});
```

**Siehe [docs/CORTEX_NEUTRON_ALTERNATIVE.md](docs/CORTEX_NEUTRON_ALTERNATIVE.md) für vollständige Migrations-Anleitung.**

## Neutron-Kompatibilität

Cortex bietet eine **vollständig neutron-kompatible Seeds-API** mit semantischer Suche:

- ✅ Gleiche Endpunkte (`/seeds`, `/seeds/query`, `/seeds/:id`)
- ✅ Gleiche Request/Response-Formate
- ✅ Multi-Tenant-Support (`appId`, `externalUserId`)
- ✅ **Semantische Suche**: Vector-Embeddings mit Cosine-Similarity
- ✅ **Lokale Embeddings**: 384-dimensionale Vektoren, vollständig offline

**Unterschiede zu Neutron:**

- 🏠 **Deployment**: Lokal (Self-hosted) statt Cloud (SaaS)
- 💰 **Kosten**: Kostenlos statt Pay-per-use
- 🔒 **Privacy**: 100% lokale Datenhaltung statt Cloud-Daten
- 📊 **Datenbank**: SQLite statt PostgreSQL + pgvector
- ⚡ **Skalierung**: Ideal für <10,000 Memories, Neutron für Enterprise-Skalierung

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

Es gibt keine API-Key-Authentifizierung; alle Endpunkte sind ohne Auth erreichbar (typisch für lokale Self-hosted-Nutzung).

### Logging

Cortex verwendet strukturiertes Logging (log/slog):

- **Log-Level:** Über `CORTEX_LOG_LEVEL` konfigurierbar (debug, info, warn, error)
- **Strukturiert:** Alle Logs enthalten strukturierte Felder für besseres Parsing
- **Format:** Text-Format (kann zu JSON geändert werden)

**Beispiel-Logs:**
```
level=INFO msg="cortex server starting" addr=:9123 db=/path/to/cortex.db
level=DEBUG msg="request" path=/seeds method=POST
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
make docker-build
# bzw. docker build -t cortex .

# Mit docker-compose starten (Port 9123)
make docker-up
# bzw. docker compose up -d
```

**Hinweis:** Wenn Port 9123 bereits belegt ist (z. B. durch einen lokal laufenden Cortex), zuerst den Prozess beenden (`pkill -f cortex`) oder in `docker-compose.yml` einen anderen Host-Port verwenden (z. B. `"9124:9123"`).

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

Wenn **lokal** ein anderer Port genutzt werden soll:

```bash
CORTEX_PORT=9124 go run ./...
```

Wenn **Docker** den Port 9123 nicht binden kann (`address already in use`): Lokalen Cortex beenden (`pkill -f cortex`) oder in `docker-compose.yml` z. B. `ports: - "9124:9123"` eintragen und Clients auf `http://localhost:9124` zeigen.

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
