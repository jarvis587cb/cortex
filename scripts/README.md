# Cortex Scripts

Hilfsscripts für die Cortex Memory API.

## Scripts

### `cortex-cli.sh` – CLI-Tool

Bash-CLI für die Cortex-API.

**Verwendung:**
```bash
./scripts/cortex-cli.sh <command> [args...]
```

**Befehle:**
- `health` – Prüft API-Status
- `store <content> [metadata]` – Speichert ein Memory
- `query <text> [limit]` – Sucht nach Memories (Standard: 5)
- `delete <id>` – Löscht ein Memory
- `stats` – Zeigt Statistiken
- `help` – Zeigt Hilfe

**Umgebungsvariablen:**
- `CORTEX_API_URL` – API Base URL (Standard: `http://localhost:9123`)
- `CORTEX_APP_ID` – App-ID für Multi-Tenant (Standard: `openclaw`)
- `CORTEX_USER_ID` – User-ID für Multi-Tenant (Standard: `default`)

**Beispiele:**
```bash
# Health Check
./scripts/cortex-cli.sh health

# Memory speichern
./scripts/cortex-cli.sh store "Der Nutzer mag Kaffee"
./scripts/cortex-cli.sh store "Präferenz" '{"tags":["preference"]}'

# Memory-Suche
./scripts/cortex-cli.sh query "Kaffee" 10

# Memory löschen
./scripts/cortex-cli.sh delete 1

# Statistiken
./scripts/cortex-cli.sh stats
```

### `benchmark.sh` – Performance-Benchmark

Misst die Performance der Cortex-API-Endpunkte.

**Verwendung:**
```bash
./scripts/benchmark.sh [anzahl]
```

**Beispiel:**
```bash
# 50 Requests pro Endpunkt
./scripts/benchmark.sh 50
```

**Output:**
```
Benchmark Ergebnisse (N=50, API=http://localhost:9123)
==========================================
health n=50 avg=0.0023s min=0.0015s max=0.0050s
store  n=50 avg=0.0125s min=0.0080s max=0.0250s
query  n=50 avg=0.0150s min=0.0100s max=0.0300s
delete n=50 avg=0.0100s min=0.0070s max=0.0200s
```

### `test-e2e.sh` – End-to-End Tests

Testet den gesamten Workflow der Cortex-API.

**Verwendung:**
```bash
./scripts/test-e2e.sh
```

**Tests:**
- Health Check
- Memory speichern
- Memory-Suche
- Memory löschen
- Statistiken

**Output:**
```
=== Health Check ===
✓ Health Check erfolgreich
=== Memory speichern ===
✓ Memory gespeichert (ID: 1)
...
==========================================
Test-Zusammenfassung
==========================================
Gesamt: 5
✓ Bestanden: 5
✓ Alle Tests bestanden!
```

## Dependencies

- `curl` – HTTP-Requests
- `jq` – JSON-Verarbeitung (optional, aber empfohlen)

**Installation:**
```bash
# Ubuntu/Debian
sudo apt-get install curl jq

# macOS
brew install curl jq
```

## Common Library

Die Scripts nutzen gemeinsame Funktionen aus `lib/common.sh`:
- Logging (`log_info`, `log_success`, `log_error`)
- HTTP-Helpers (`curl_with_status`, `parse_http_response`)
- JSON-Helpers (`format_json`, `extract_id`, `count_items`)
- Validierung (`is_positive_integer`, `has_jq`)
