# Cortex API Dokumentation

Vollständige API-Referenz für Cortex Memory API - Neutron-kompatibel.

## Inhaltsverzeichnis

- [Authentifizierung](#authentifizierung)
- [Basis-URL](#basis-url)
- [Neutron-kompatible Seeds API](#neutron-kompatible-seeds-api)
- [Bundles API](#bundles-api)
- [Cortex API](#cortex-api)
- [Fehlerbehandlung](#fehlerbehandlung)
- [Beispiele](#beispiele)

## Authentifizierung

Alle Endpunkte (außer `/health`) erfordern Authentifizierung über den `X-API-Key` Header:

```http
X-API-Key: dein-api-key
```

**Hinweis:** Es gibt keine API-Key-Authentifizierung; alle Endpunkte sind ohne Auth erreichbar.

## Basis-URL

Standard: `http://localhost:9123`

Konfigurierbar über Umgebungsvariable `CORTEX_PORT`.

## Neutron-kompatible Seeds API

Vollständig kompatibel mit Neutron Memory API. Unterstützt beide Parameter-Formate:

- **Query-Parameter** (Neutron-Style): `?appId=xxx&externalUserId=yyy`
- **Body-Parameter** (Cortex-Style): `{ "appId": "xxx", "externalUserId": "yyy" }`

### `POST /seeds` - Memory speichern

Speichert ein neues Memory (Seed) und generiert automatisch ein Embedding.

**Query-Parameter (optional, Neutron-Style):**
- `appId` (string, erforderlich)
- `externalUserId` (string, erforderlich)

**Request Body:**
```json
{
  "appId": "myapp",                    // Optional wenn im Query-String
  "externalUserId": "user123",         // Optional wenn im Query-String
  "content": "Der Benutzer mag Kaffee",
  "metadata": {                        // Optional
    "source": "chat",
    "tags": ["preferences"]
  },
  "bundleId": 1                        // Optional: Bundle-ID
}
```

**Response (200 OK):**
```json
{
  "id": 42,
  "message": "Memory stored successfully"
}
```

**Beispiele:**

```bash
# Mit Query-Parametern (Neutron-Style)
curl -X POST "http://localhost:9123/seeds?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "content": "Der Benutzer mag Kaffee",
    "metadata": {"source": "chat"}
  }'

# Mit Body-Parametern (Cortex-Style)
curl -X POST http://localhost:9123/seeds \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "appId": "myapp",
    "externalUserId": "user123",
    "content": "Der Benutzer mag Kaffee",
    "metadata": {"source": "chat"}
  }'
```

### `POST /seeds/query` - Memory-Suche

Führt semantische Suche durch (mit Embeddings) oder fällt auf Textsuche zurück.

**Query-Parameter (optional, Neutron-Style):**
- `appId` (string, erforderlich)
- `externalUserId` (string, erforderlich)

**Request Body:**
```json
{
  "appId": "myapp",                    // Optional wenn im Query-String
  "externalUserId": "user123",         // Optional wenn im Query-String
  "query": "Was mag der Benutzer trinken?",
  "limit": 5,                          // Optional, Standard: 5, Max: 100
  "bundleId": 1                        // Optional: Filter nach Bundle
}
```

**Response (200 OK):**
```json
[
  {
    "id": 42,
    "content": "Der Benutzer mag Kaffee",
    "metadata": {"source": "chat"},
    "created_at": "2026-02-19T10:30:00Z",
    "similarity": 0.95
  },
  {
    "id": 38,
    "content": "Der Benutzer trinkt gerne Tee",
    "metadata": {"source": "chat"},
    "created_at": "2026-02-19T09:15:00Z",
    "similarity": 0.82
  }
]
```

**Beispiele:**

```bash
# Mit Query-Parametern (Neutron-Style)
curl -X POST "http://localhost:9123/seeds/query?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "query": "Was mag der Benutzer?",
    "limit": 5
  }'

# Mit Body-Parametern (Cortex-Style)
curl -X POST http://localhost:9123/seeds/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "appId": "myapp",
    "externalUserId": "user123",
    "query": "Was mag der Benutzer?",
    "limit": 5
  }'
```

### `DELETE /seeds/:id` - Memory löschen

Löscht ein Memory anhand der ID.

**Query-Parameter (erforderlich):**
- `appId` (string)
- `externalUserId` (string)

**Response (200 OK):**
```json
{
  "message": "Memory deleted successfully",
  "id": 42
}
```

**Beispiel:**

```bash
curl -X DELETE "http://localhost:9123/seeds/42?appId=myapp&externalUserId=user123" \
  -H "X-API-Key: dein-key"
```

### `POST /seeds/generate-embeddings` - Embeddings batch-generieren

Generiert Embeddings für alle Memories ohne Embedding.

**Query-Parameter (optional):**
- `batchSize` (int, Standard: 10, Max: 100)

**Response (200 OK):**
```json
{
  "message": "Embeddings generation started"
}
```

**Beispiel:**

```bash
curl -X POST "http://localhost:9123/seeds/generate-embeddings?batchSize=20" \
  -H "X-API-Key: dein-key"
```

## Bundles API

Bundles ermöglichen die Organisation von Memories in logische Gruppen.

### `POST /bundles` - Bundle erstellen

Erstellt ein neues Bundle.

**Query-Parameter (optional, Neutron-Style):**
- `appId` (string, erforderlich)
- `externalUserId` (string, erforderlich)

**Request Body:**
```json
{
  "appId": "myapp",                    // Optional wenn im Query-String
  "externalUserId": "user123",         // Optional wenn im Query-String
  "name": "Coffee Preferences"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "Coffee Preferences",
  "app_id": "myapp",
  "external_user_id": "user123",
  "created_at": "2026-02-19T10:30:00Z"
}
```

**Beispiel:**

```bash
curl -X POST "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{"name": "Coffee Preferences"}'
```

### `GET /bundles` - Bundles auflisten

Listet alle Bundles für einen Tenant auf.

**Query-Parameter (erforderlich):**
- `appId` (string)
- `externalUserId` (string)

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "name": "Coffee Preferences",
    "app_id": "myapp",
    "external_user_id": "user123",
    "created_at": "2026-02-19T10:30:00Z"
  },
  {
    "id": 2,
    "name": "Reading Habits",
    "app_id": "myapp",
    "external_user_id": "user123",
    "created_at": "2026-02-19T09:15:00Z"
  }
]
```

**Beispiel:**

```bash
curl "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
  -H "X-API-Key: dein-key"
```

### `GET /bundles/:id` - Bundle abrufen

Ruft ein Bundle anhand der ID ab.

**Query-Parameter (erforderlich):**
- `appId` (string)
- `externalUserId` (string)

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "Coffee Preferences",
  "app_id": "myapp",
  "external_user_id": "user123",
  "created_at": "2026-02-19T10:30:00Z"
}
```

**Beispiel:**

```bash
curl "http://localhost:9123/bundles/1?appId=myapp&externalUserId=user123" \
  -H "X-API-Key: dein-key"
```

### `DELETE /bundles/:id` - Bundle löschen

Löscht ein Bundle. **Hinweis:** Memories bleiben erhalten, `bundleId` wird auf `NULL` gesetzt.

**Query-Parameter (erforderlich):**
- `appId` (string)
- `externalUserId` (string)

**Response (200 OK):**
```json
{
  "message": "Bundle deleted successfully",
  "id": 1
}
```

**Beispiel:**

```bash
curl -X DELETE "http://localhost:9123/bundles/1?appId=myapp&externalUserId=user123" \
  -H "X-API-Key: dein-key"
```

## Cortex API

Zusätzliche Endpunkte für erweiterte Funktionalität.

### `POST /remember` - Erinnerung speichern

Speichert eine Erinnerung (Cortex-spezifisches Format).

**Request Body:**
```json
{
  "content": "Wichtige Erinnerung",
  "type": "semantic",                 // Optional, Standard: "semantic"
  "entity": "user:alice",              // Optional
  "tags": "important,meeting",         // Optional
  "importance": 7                      // Optional, Standard: 5, Range: 1-10
}
```

**Response (200 OK):**
```json
{
  "id": 42
}
```

### `GET /recall` - Erinnerungen abrufen

Ruft Erinnerungen ab (mit optionalen Filtern).

**Query-Parameter:**
- `query` (string, optional) - Suchbegriff
- `type` (string, optional) - Filter nach Typ
- `limit` (int, optional, Standard: 10)

**Response (200 OK):**
```json
[
  {
    "id": 42,
    "type": "semantic",
    "content": "Wichtige Erinnerung",
    "entity": "user:alice",
    "tags": "important,meeting",
    "importance": 7,
    "created_at": "2026-02-19T10:30:00Z"
  }
]
```

### `GET /entities` - Entities auflisten

Listet alle Entities auf.

**Query-Parameter:**
- `name` (string, optional) - Filter nach Name

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "name": "user:alice",
    "data": {"key": "value"},
    "created_at": "2026-02-19T10:30:00Z",
    "updated_at": "2026-02-19T10:30:00Z"
  }
]
```

### `POST /entities` - Entity erstellen/aktualisieren

Erstellt oder aktualisiert eine Entity.

**Request Body:**
```json
{
  "key": "user:alice",
  "value": {"name": "Alice", "age": 30}
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "user:alice",
  "data": {"name": "Alice", "age": 30}
}
```

### `GET /relations` - Relationen auflisten

Listet alle Relationen auf.

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "from": "user:alice",
    "to": "user:bob",
    "type": "friend",
    "created_at": "2026-02-19T10:30:00Z"
  }
]
```

### `POST /relations` - Relation hinzufügen

Fügt eine Relation zwischen Entities hinzu.

**Request Body:**
```json
{
  "from": "user:alice",
  "to": "user:bob",
  "type": "friend"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "from": "user:alice",
  "to": "user:bob",
  "type": "friend"
}
```

### `GET /stats` - Statistiken abrufen

Ruft Statistiken über die Datenbank ab.

**Response (200 OK):**
```json
{
  "memories": 42,
  "entities": 5,
  "relations": 12
}
```

### `GET /health` - Health Check

Prüft den API-Status. **Keine Authentifizierung erforderlich.**

**Response (200 OK):**
```json
{
  "status": "ok",
  "timestamp": "2026-02-19T10:30:00Z"
}
```

## Fehlerbehandlung

### HTTP-Status-Codes

- `200 OK` - Erfolgreiche Anfrage
- `400 Bad Request` - Ungültige Anfrage (fehlende Parameter, ungültiges JSON)
- `401 Unauthorized` - Authentifizierung fehlgeschlagen
- `404 Not Found` - Ressource nicht gefunden
- `405 Method Not Allowed` - HTTP-Methode nicht erlaubt
- `500 Internal Server Error` - Server-Fehler

### Fehler-Response-Format

```json
{
  "error": "Fehlerbeschreibung",
  "message": "Detaillierte Fehlermeldung"
}
```

**Beispiele:**

```json
// 400 Bad Request
{
  "error": "missing required field: appId"
}

// 404 Not Found
{
  "error": "Memory not found"
}

// 500 Internal Server Error
{
  "error": "internal error"
}
```

## Beispiele

### Vollständiges Beispiel: Memory mit Bundle

```bash
# 1. Bundle erstellen
BUNDLE_ID=$(curl -s -X POST "http://localhost:9123/bundles?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{"name": "Coffee Preferences"}' | jq -r '.id')

# 2. Memory in Bundle speichern
MEMORY_ID=$(curl -s -X POST "http://localhost:9123/seeds?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d "{
    \"content\": \"Lieblingskaffee: Latte mit Hafermilch\",
    \"metadata\": {\"source\": \"chat\"},
    \"bundleId\": $BUNDLE_ID
  }" | jq -r '.id')

# 3. Memories in Bundle suchen
curl -X POST "http://localhost:9123/seeds/query?appId=myapp&externalUserId=user123" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d "{
    \"query\": \"Kaffee\",
    \"bundleId\": $BUNDLE_ID,
    \"limit\": 10
  }"
```

### TypeScript SDK Beispiel

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

## Rate Limits

Cortex unterstützt **Token-Bucket Rate Limiting** zur Begrenzung der API-Anfragen.

### Konfiguration

**Umgebungsvariablen:**
- `CORTEX_RATE_LIMIT` – Anzahl der erlaubten Requests pro Zeitfenster (Standard: 100)
- `CORTEX_RATE_LIMIT_WINDOW` – Zeitfenster für Rate Limiting (Standard: `1m`)

**Beispiele:**
```bash
# 100 Requests pro Minute (Standard)
export CORTEX_RATE_LIMIT=100
export CORTEX_RATE_LIMIT_WINDOW=1m

# 1000 Requests pro Stunde
export CORTEX_RATE_LIMIT=1000
export CORTEX_RATE_LIMIT_WINDOW=1h

# Rate Limiting deaktivieren
export CORTEX_RATE_LIMIT=0
```

### Verhalten

- **Client-Identifikation:** Basierend auf API-Key oder IP-Adresse
- **Token-Bucket:** Proportionale Token-Auffüllung über Zeitfenster
- **Response:** `429 Too Many Requests` mit `Retry-After` Header
- **Health-Check:** `/health` Endpunkt ist von Rate Limiting ausgenommen

### Beispiel-Response

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
Content-Type: text/plain

rate limit exceeded
```

## Versionierung

Aktuelle API-Version: **v1**

Keine Versions-Präfixe in URLs. Breaking Changes werden durch neue Endpunkte oder Parameter gehandhabt.

## Webhooks

Cortex unterstützt **Webhooks** für Event-Benachrichtigungen.

### Events

Verfügbare Event-Typen:
- `memory.created` – Memory wurde erstellt
- `memory.deleted` – Memory wurde gelöscht
- `bundle.created` – Bundle wurde erstellt
- `bundle.deleted` – Bundle wurde gelöscht

### Webhook erstellen

```bash
curl -X POST http://localhost:9123/webhooks \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d '{
    "url": "https://example.com/webhook",
    "events": ["memory.created", "memory.deleted"],
    "secret": "webhook-secret",
    "appId": "myapp"
  }'
```

**Response:**
```json
{
  "id": 1,
  "url": "https://example.com/webhook",
  "events": ["memory.created", "memory.deleted"],
  "app_id": "myapp",
  "active": true,
  "created_at": "2026-02-19T10:30:00Z",
  "updated_at": "2026-02-19T10:30:00Z"
}
```

### Webhooks auflisten

```bash
curl "http://localhost:9123/webhooks?appId=myapp" \
  -H "X-API-Key: dein-key"
```

### Webhook löschen

```bash
curl -X DELETE "http://localhost:9123/webhooks/1" \
  -H "X-API-Key: dein-key"
```

### Webhook-Payload

**Format:**
```json
{
  "event": "memory.created",
  "timestamp": "2026-02-19T10:30:00Z",
  "data": {
    "id": 42,
    "app_id": "myapp",
    "external_user_id": "user123",
    "content": "Der Benutzer mag Kaffee",
    "bundle_id": 1,
    "created_at": "2026-02-19T10:30:00Z"
  }
}
```

### Webhook-Signatur

Wenn ein `secret` konfiguriert ist, wird jeder Webhook mit HMAC-SHA256 signiert:

**Header:**
```
X-Cortex-Signature: sha256=<signature>
```

**Verifikation:**
```javascript
const crypto = require('crypto');

function verifySignature(secret, payload, signature) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(payload);
  const expected = 'sha256=' + hmac.digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}
```

### Webhook-Delivery

- **Asynchron:** Webhooks werden asynchron ausgeliefert (nicht-blockierend)
- **Timeout:** 10 Sekunden pro Webhook
- **Retry:** Keine automatischen Retries (kann in Zukunft hinzugefügt werden)
- **Filterung:** Nur aktive Webhooks mit passendem Event-Typ werden ausgelöst
- **App-Filter:** Webhooks können app-spezifisch sein (`appId`) oder global

## Export/Import

Cortex unterstützt **Export und Import** von Daten für Migration und Backup.

### `GET /export` - Daten exportieren

Exportiert alle Daten (Memories, Bundles, Webhooks) für einen Tenant als JSON.

**Query-Parameter (erforderlich):**
- `appId` (string)
- `externalUserId` (string)

**Response (200 OK):**
```json
{
  "version": "1.0",
  "export_date": "2026-02-19T10:30:00Z",
  "memories": [...],
  "bundles": [...],
  "webhooks": [...]
}
```

**Beispiel:**

```bash
curl "http://localhost:9123/export?appId=myapp&externalUserId=user123" \
  -H "X-API-Key: dein-key" \
  -o cortex-export.json
```

### `POST /import` - Daten importieren

Importiert Daten aus einem Export-File.

**Query-Parameter (erforderlich):**
- `appId` (string)
- `externalUserId` (string)
- `overwrite` (boolean, optional) - Wenn `true`, werden existierende Einträge überschrieben

**Request Body:**
```json
{
  "version": "1.0",
  "export_date": "2026-02-19T10:30:00Z",
  "memories": [...],
  "bundles": [...],
  "webhooks": [...]
}
```

**Response (200 OK):**
```json
{
  "message": "Import completed successfully",
  "memories": 42,
  "bundles": 5,
  "webhooks": 2,
  "overwrite": false
}
```

**Beispiel:**

```bash
curl -X POST "http://localhost:9123/import?appId=myapp&externalUserId=user123&overwrite=false" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dein-key" \
  -d @cortex-export.json
```

## Backup/Restore

Cortex unterstützt **Backup und Restore** der gesamten Datenbank.

### `POST /backup` - Datenbank-Backup erstellen

Erstellt ein Backup der SQLite-Datenbank.

**Query-Parameter (optional):**
- `path` (string) - Pfad für Backup-Datei (Standard: `cortex-backup-YYYYMMDD-HHMMSS.db`)

**Response (200 OK):**
```json
{
  "message": "Backup created successfully",
  "path": "cortex-backup-20260219-103000.db"
}
```

**Beispiel:**

```bash
# Backup mit Standard-Pfad
curl -X POST "http://localhost:9123/backup" \
  -H "X-API-Key: dein-key"

# Backup mit benutzerdefiniertem Pfad
curl -X POST "http://localhost:9123/backup?path=/backups/cortex-backup.db" \
  -H "X-API-Key: dein-key"
```

### `POST /restore` - Datenbank wiederherstellen

Stellt die Datenbank aus einem Backup wieder her.

**⚠️ WICHTIG:** Nach dem Restore muss der Server neu gestartet werden!

**Query-Parameter (erforderlich):**
- `path` (string) - Pfad zur Backup-Datei

**Response (200 OK):**
```json
{
  "message": "Restore completed successfully. Server restart required to use restored database.",
  "backup_path": "/backups/cortex-backup.db",
  "restored_to": "/path/to/cortex.db",
  "warning": "Server must be restarted for changes to take effect"
}
```

**Beispiel:**

```bash
curl -X POST "http://localhost:9123/restore?path=/backups/cortex-backup.db" \
  -H "X-API-Key: dein-key"
```

**Hinweis:** Der Restore-Prozess kopiert die Backup-Datei über die aktuelle Datenbank. Ein Server-Neustart ist erforderlich, damit die Änderungen wirksam werden.

## Analytics

Cortex bietet **Analytics-Endpunkte** für Dashboard-Daten und Metriken.

### `GET /analytics` - Analytics-Daten abrufen

Ruft Analytics-Daten für einen Tenant oder global ab.

**Query-Parameter:**
- `appId` (string, optional) - Für Tenant-spezifische Analytics
- `externalUserId` (string, optional) - Für Tenant-spezifische Analytics
- `days` (int, optional) - Anzahl der Tage für Zeitraum (Standard: 30, Max: 365)

**Response (200 OK):**
```json
{
  "tenant_id": "myapp:user123",
  "app_id": "myapp",
  "external_user_id": "user123",
  "total_memories": 42,
  "total_bundles": 5,
  "memories_with_embeddings": 38,
  "memories_by_type": {
    "semantic": 35,
    "episodic": 7
  },
  "memories_by_bundle": {
    "1": 10,
    "2": 5
  },
  "recent_activity": [
    {
      "type": "memory.created",
      "id": 42,
      "timestamp": "2026-02-19T10:30:00Z"
    }
  ],
  "storage_stats": {
    "memories_count": 42,
    "bundles_count": 5,
    "webhooks_count": 2
  },
  "time_range": {
    "start": "2026-01-20T10:30:00Z",
    "end": "2026-02-19T10:30:00Z"
  }
}
```

**Beispiele:**

```bash
# Tenant-spezifische Analytics (letzte 30 Tage)
curl "http://localhost:9123/analytics?appId=myapp&externalUserId=user123" \
  -H "X-API-Key: dein-key"

# Tenant-spezifische Analytics (letzte 7 Tage)
curl "http://localhost:9123/analytics?appId=myapp&externalUserId=user123&days=7" \
  -H "X-API-Key: dein-key"

# Globale Analytics (alle Tenants)
curl "http://localhost:9123/analytics?days=30" \
  -H "X-API-Key: dein-key"
```

**Verfügbare Metriken:**
- **Total Memories** - Gesamtanzahl der Memories
- **Total Bundles** - Gesamtanzahl der Bundles
- **Memories with Embeddings** - Anzahl der Memories mit Embeddings
- **Memories by Type** - Aufschlüsselung nach Memory-Typ
- **Memories by Bundle** - Aufschlüsselung nach Bundle
- **Recent Activity** - Letzte 50 Aktivitäten (Memories/Bundles erstellt)
- **Storage Stats** - Speicher-Statistiken

## Support

- **Dokumentation:** Siehe [README.md](README.md)
- **SDK:** Siehe [sdk/README.md](sdk/README.md)
- **Issues:** GitHub Issues
