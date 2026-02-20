# Cortex Scripts

Hilfsscripts für die Cortex Memory API.

## Scripts

### `cortex-memory.sh` – Neutron-kompatibles Script (OpenClaw-Guide)

Gleiche Befehle wie im [OpenClaw Integration Guide](https://openclaw.vanarchain.com/guide-openclaw) (Neutron): `test`, `save`, `search`, **`recall`**, **`capture`** (Hooks), `context-create`, `context-list`, `context-get`. Nutzt Cortex-API (kein API-Key).

**Verwendung:**
```bash
./scripts/cortex-memory.sh test
./scripts/cortex-memory.sh save "content" "[metadata_json]"
./scripts/cortex-memory.sh search "query" [limit] [threshold] [seedIds]
./scripts/cortex-memory.sh recall "[query]" [limit] [threshold]   # Hook: vor Interaktion (VANAR_AUTO_RECALL)
./scripts/cortex-memory.sh capture "content" [metadata_json]      # Hook: nach Austausch (VANAR_AUTO_CAPTURE)
./scripts/cortex-memory.sh context-create "agentId" [memoryType] [payload_json]
./scripts/cortex-memory.sh context-list [agentId]
./scripts/cortex-memory.sh context-get <id>
```

**Umgebungsvariablen:** `CORTEX_API_URL`, `CORTEX_APP_ID`, `CORTEX_USER_ID` (oder `NEUTRON_*`); optional `CORTEX_API_KEY`; für Hooks: `VANAR_AUTO_RECALL`, `VANAR_AUTO_CAPTURE` (default: true).

Siehe [skills/cortex/SKILL.md](../skills/cortex/SKILL.md) für Details und **hooks.sh** (`skills/cortex/hooks.sh`, Einstiegspunkt für OpenClaw).

### `api-key.sh` – API-Key anlegen / löschen

Verwaltet `CORTEX_API_KEY` in einer `.env`-Datei (Server oder Client).

```bash
./scripts/api-key.sh create [env_file]   # Neuen Key erzeugen und in .env setzen
./scripts/api-key.sh delete [env_file]    # Key aus .env entfernen
./scripts/api-key.sh show [env_file]     # Anzeigen, ob Key gesetzt (nur letzte 4 Zeichen)
```

Ohne `env_file` wird die `.env` im Projektroot oder im aktuellen Verzeichnis verwendet. Der Key hat das Format `ck_` + 64 Hex-Zeichen. Nach `create` Server neu starten bzw. Variable exportieren.

### `cortex-cli.sh` – CLI-Tool (Bash)

Bash-CLI für die Cortex-API. **Alternative:** Die Go-Binary `cortex-cli` (nach `make build` als `./cortex-cli` verfügbar) bietet die gleichen Befehle ohne Abhängigkeit von jq/curl; siehe Haupt-[README](../README.md).

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
