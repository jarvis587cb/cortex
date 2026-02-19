# API Tests

Dieses Verzeichnis enthält Test-Dateien für die Cortex API.

## Dateien

### `api.http`
HTTP-Datei für VS Code REST Client Extension oder IntelliJ HTTP Client.

**Verwendung:**
1. Installiere die REST Client Extension in VS Code
2. Öffne `api.http`
3. Klicke auf "Send Request" über jeder Anfrage

**Variablen anpassen:**
- `@baseUrl` - API Base URL (Standard: http://localhost:9123)
- `@appId` - App-ID für Multi-Tenant (Standard: openclaw)
- `@userId` - User-ID für Multi-Tenant (Standard: default)

### `curl-examples.sh`
Bash-Script mit curl-Beispielen für alle API-Endpunkte.

**Verwendung:**
```bash
# Standard
./tests/curl-examples.sh

# Mit angepassten Werten
CORTEX_API_URL=http://localhost:9123 \
CORTEX_APP_ID=my-app \
CORTEX_USER_ID=my-user \
./tests/curl-examples.sh
```

**Voraussetzungen:**
- `curl` installiert
- `jq` installiert (optional, für JSON-Formatierung)
- Laufender Cortex-Server

## Test-Szenarien

### 1. Health Check
- Endpunkt: `GET /health`
- Keine Authentifizierung erforderlich
- Testet Server-Verfügbarkeit

### 2. Neutron-kompatible Seeds API
- `POST /seeds` - Memory speichern
- `POST /seeds/query` - Memory-Suche
- `DELETE /seeds/:id` - Memory löschen

### 3. Cortex API (Original)
- `POST /remember` - Erinnerung speichern
- `GET /recall` - Erinnerungen abrufen
- `POST /entities` - Fakt setzen
- `GET /entities` - Entity abrufen
- `POST /relations` - Relation hinzufügen
- `GET /relations` - Relationen abrufen
- `GET /stats` - Statistiken

### 4. Fehler-Tests
- Ungültige Requests
- Fehlende Parameter
- Ungültige Methoden

## Server starten

Vor dem Testen muss der Server laufen:

```bash
# Lokal
make run

# Oder mit Docker
make docker-run
```

## Beispiel-Ausführung

```bash
# 1. Server starten
make run

# 2. In neuem Terminal: Tests ausführen
cd tests
./curl-examples.sh

# 3. Oder HTTP-Datei in VS Code öffnen
code api.http
```
