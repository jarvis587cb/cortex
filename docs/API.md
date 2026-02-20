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

**Optional:** API-Key-Authentifizierung ist nur für Produktions-/Multi-User-Setups erforderlich. **Für lokale Installationen ist kein API-Key nötig.**

Wenn die Umgebungsvariable `CORTEX_API_KEY` am Server gesetzt ist, müssen alle Anfragen (außer `GET /health`) einen gültigen Key mitsenden. `/health` bleibt ohne Auth (für Load-Balancer und Checks).

- **Header:** `Authorization: Bearer <CORTEX_API_KEY>` oder `X-API-Key: <CORTEX_API_KEY>`
- **Lokal/Dev:** Wenn `CORTEX_API_KEY` leer ist (Standard für lokale Installationen), sind alle Endpunkte ohne Auth erreichbar (Neutron-kompatibel: gleiche Header wie im [OpenClaw Guide](https://openclaw.vanarchain.com/guide-openclaw)).

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

**CLI:**
```bash
cortex-cli store "Der Benutzer mag Kaffee" '{"source":"chat"}'
# App/User aus -app-id/-user-id oder CORTEX_APP_ID/CORTEX_USER_ID
```

### `GET /seeds` - Memories auflisten (Pagination)

Gibt eine paginierte Liste von Memories für einen Tenant zurück (z. B. für das Dashboard). Embeddings werden nicht mitgeliefert.

**Query-Parameter:**
- `appId` (string, erforderlich)
- `externalUserId` (string, erforderlich)
- `limit` (int, optional) – Standard 50, Max 100
- `offset` (int, optional) – Standard 0

**Response (200 OK):**
JSON-Array von Memory-Objekten (id, content, type, metadata, created_at usw., ohne embedding).

**CLI:**
```bash
cortex-cli seeds-list 20 0
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
  "threshold": 0.5,                    // Optional, 0-1: nur Ergebnisse mit similarity >= threshold
  "bundleId": 1,                        // Optional: Filter nach Bundle
  "seedIds": [1, 2, 3],                // Optional: limit search to these memory IDs
  "metadataFilter": {                  // Optional: filter by metadata fields
    "typ": "persönlich",
    "kategorie": "präferenz"
  }
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

**CLI:**
```bash
cortex-cli query "Was mag der Benutzer?" 5 0.5
cortex-cli query "Theme" 10 0.5 "" '{"typ":"persönlich"}'
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

**CLI:**
```bash
cortex-cli delete 42
```

### `POST /seeds/generate-embeddings` - Embeddings batch-generieren

Generiert Embeddings für alle Memories ohne Embedding.

**Query-Parameter (optional):**
- `batchSize` (int, Standard: 10, Max: 100)

**Response (200 OK):**
```json
{
  "message": "Embeddings generation completed"
}
```

**CLI:**
```bash
cortex-cli generate-embeddings 20
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

**CLI:**
```bash
cortex-cli bundle-create "Coffee Preferences"
# Gibt die neue Bundle-ID aus (für Scripts: BUNDLE_ID=$(cortex-cli bundle-create "Coffee Preferences"))
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

**CLI:**
```bash
cortex-cli bundle-list
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

**CLI:**
```bash
cortex-cli bundle-get 1
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

**CLI:**
```bash
cortex-cli bundle-delete 1
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

### Vollständiges Beispiel: Memory mit Bundle (CLI)

```bash
# 1. Bundle erstellen (CLI gibt die ID aus)
BUNDLE_ID=$(cortex-cli bundle-create "Coffee Preferences")

# 2. Memory in Bundle speichern (store unterstützt kein bundleId direkt – über API oder SDK)
cortex-cli store "Lieblingskaffee: Latte mit Hafermilch" '{"source":"chat"}'

# 3. Memories suchen (Query mit optionalem bundleId über SDK/API; CLI: query "Kaffee" 10)
cortex-cli query "Kaffee" 10
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

- **Client-Identifikation:** Basierend auf Request-Header (falls gesetzt) oder IP-Adresse
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

**CLI:**
```bash
cortex-cli webhook-create "https://example.com/webhook" "memory.created,memory.deleted" "webhook-secret"
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

**CLI:**
```bash
cortex-cli webhook-list
```

### Webhook löschen

`appId` (Query) ist erforderlich (Tenant-Isolation).

**CLI:**
```bash
cortex-cli webhook-delete 1
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

**CLI:**
```bash
cortex-cli export cortex-export.json
# Ohne Dateiname: Ausgabe auf stdout
cortex-cli export
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

**CLI:**
```bash
cortex-cli import cortex-export.json
# Mit Überschreiben existierender Daten:
cortex-cli import cortex-export.json true
# Von stdin: cortex-cli import -
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

**CLI:**
```bash
cortex-cli backup
cortex-cli backup /backups/cortex-backup.db
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

**CLI:**
```bash
cortex-cli restore /backups/cortex-backup.db
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

**CLI:**
```bash
cortex-cli analytics           # Letzte 30 Tage (Standard)
cortex-cli analytics 7         # Letzte 7 Tage
# Ohne appId/userId (global): nur über HTTP-API mit weggelassenen Query-Parametern
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
