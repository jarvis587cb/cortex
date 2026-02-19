# Cortex – Gehirn-Backend für OpenClaw

Cortex ist ein **leichtgewichtiges Go-Backend** mit SQLite-Datenbank, das als persistentes „Gehirn“ für OpenClaw-Agenten dient. Es speichert Erinnerungen (Memories), Entities mit Fakten sowie Relationen zwischen Entities und wird über ein OpenClaw-Plugin angebunden.

## Features

- ✅ **Persistente Speicherung**: Erinnerungen, Fakten und Relationen in SQLite
- ✅ **Multi-Tenant-Support**: Isolation durch `appId` + `externalUserId`
- ✅ **Neutron-kompatibel**: Gleiche API-Formate wie neutron-local (ohne Embeddings)
- ✅ **Leichtgewichtig**: Pure-Go (kein cgo), keine externen Dependencies außer SQLite
- ✅ **REST-API**: Einfache HTTP-Endpunkte für alle Operationen
- ✅ **OpenClaw-Integration**: TypeScript-Plugin mit Agent-Tools

## Architektur

Cortex besteht aus zwei Komponenten:

### 1. Go-Server (`cortex`)

**Backend-Service** mit SQLite-Datenbank und HTTP-API:

- **Datenbank**: SQLite (`~/.openclaw/cortex.db` oder über `CORTEX_DB_PATH`)
- **Port**: 9123 (Standard) oder über `CORTEX_PORT`
- **Technologie**: Go 1.23+, GORM, `github.com/glebarez/sqlite` (pure-Go)

### 2. OpenClaw-Plugin (`plugin/`)

**TypeScript-Plugin** für OpenClaw-Agenten:

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

**Health-Check:**

```bash
curl http://localhost:9123/health
# {"status":"ok","timestamp":"2026-02-19T15:00:00Z"}
```

### OpenClaw-Plugin installieren

1. **Plugin lokal laden:**

   ```bash
   openclaw plugins install -l /home/jarvis/.openclaw/workspace/projects/cortex/plugin
   ```

2. **Dependencies installieren** (falls nötig):

   ```bash
   cd plugin
   npm install
   ```

3. **Config in `~/.openclaw/openclaw.json`:**

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

4. **Tools im Agent aktivieren:**

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

**Hinweis:** `similarity` ist eine Heuristik (0.8-1.0), da keine echten Embeddings verwendet werden.

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

## Agent-Tools

Das Plugin registriert folgende Tools für OpenClaw-Agenten:

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

### Build

```bash
# Go-Binary bauen
go build -o cortex .

# Plugin-Dependencies installieren
cd plugin
npm install
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

### Plugin wird nicht geladen

1. Plugin-Pfad prüfen: `openclaw plugins list`
2. Config validieren: `openclaw doctor`
3. Gateway neu starten nach Config-Änderungen

## Lizenz

MIT (oder wie im Workspace definiert)
