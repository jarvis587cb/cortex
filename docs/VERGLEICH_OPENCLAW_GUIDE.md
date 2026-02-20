# Vergleich: Cortex vs. OpenClaw Neutron Integration Guide

Referenz: [OpenClaw Integration Guide](https://openclaw.vanarchain.com/guide-openclaw) (Vanar Neutron)

## Übersicht

| Aspekt | OpenClaw Guide (Neutron) | Cortex |
|--------|--------------------------|--------|
| **Typ** | Cloud-basiert (Vanar Neutron) | Lokal (SQLite) |
| **API-Key** | ✅ Erforderlich | ❌ Nicht erforderlich (lokal) |
| **Installation** | ClawHub Skill | Lokaler Server + Skill |
| **Datenbank** | Cloud (Vanar) | SQLite (lokal) |
| **Kosten** | Abhängig von Nutzung | Kostenlos (selbst gehostet) |

## Installation & Konfiguration

### OpenClaw Guide (Neutron)

```bash
# 1. ClawHub CLI installieren
npm i -g clawhub

# 2. Skill installieren
clawhub install vanar-neutron-memory

# 3. Credentials konfigurieren (.env oder ~/.config/neutron/credentials.json)
NEUTRON_API_KEY=your_key
NEUTRON_AGENT_ID=your_agent_id
YOUR_AGENT_IDENTIFIER=your_agent_name_or_id

# 4. Testen
./scripts/neutron-memory.sh test
```

### Cortex

```bash
# 1. Repository klonen
git clone https://github.com/jarvis587cb/cortex.git
cd cortex

# 2. Binaries bauen
make build

# 3. Server starten (optional: systemd Service)
make run
# oder
make service-install
make service-enable
make service-start

# 4. Testen
./cortex-cli health
```

**Konfiguration (.env - optional):**
```bash
CORTEX_PORT=9123
CORTEX_DB_PATH=~/.openclaw/cortex.db
CORTEX_LOG_LEVEL=info
# CORTEX_API_KEY=  # Nur für Produktion/Multi-User
```

**Keine API-Key erforderlich** für lokale Installation! ✅

## Hooks (Auto-Recall / Auto-Capture)

### OpenClaw Guide (Neutron)

- **Auto-Recall**: Vor jeder AI-Interaktion werden relevante Memories abgerufen
- **Auto-Capture**: Nach jedem Austausch werden Konversationen gespeichert
- **Konfiguration**: Via `.env` oder `credentials.json`
  ```bash
  VANAR_AUTO_RECALL=false
  VANAR_AUTO_CAPTURE=false
  ```

### Cortex

**Status:** ✅ Implementiert

**Implementierung:**
- ✅ `skills/cortex/hooks.sh` für Auto-Recall/Capture
- ✅ Env-Variablen: `CORTEX_AUTO_RECALL`, `CORTEX_AUTO_CAPTURE`
- ✅ Unterstützt JSON und Plain-Text Input
- ✅ Integration in OpenClaw Agent-Hooks

**Verwendung:**
```bash
# Recall-Hook (vor AI-Interaktion)
echo '{"message": "user question"}' | hooks.sh recall

# Capture-Hook (nach Konversation)
echo '{"content": "conversation", "appId": "...", "userId": "..."}' | hooks.sh capture
```

**Konfiguration:**
```bash
CORTEX_AUTO_RECALL=true      # Default: true
CORTEX_AUTO_CAPTURE=true     # Default: true
CORTEX_API_URL=http://localhost:9123
CORTEX_APP_ID=openclaw
CORTEX_USER_ID=default
```

## Seeds (Memory Storage & Search)

### OpenClaw Guide (Neutron)

```bash
# Memory speichern
./scripts/neutron-memory.sh save "User prefers oat milk lattes" "User coffee preference"

# Memory suchen
./scripts/neutron-memory.sh search "what do I know about blockchain" 10 0.5

# Parameter:
# - query (required)
# - limit (optional, default: 30)
# - threshold (optional, default: 0.5)
# - seedIds (optional)
```

**Seed-Typen:** `text`, `json`, `markdown`, `csv`, `email`, `claude_chat`, `gpt_chat`

### Cortex

```bash
# Memory speichern
./cortex-cli store "User prefers oat milk lattes" '{"type":"fact"}'

# Memory suchen
./cortex-cli query "what do I know about blockchain" 10 0.5

# Memory löschen
./cortex-cli delete <id>

# Statistiken
./cortex-cli stats
```

**API-Endpunkte:**
- ✅ `POST /seeds` - Memory speichern
- ✅ `POST /seeds/query` - Semantische Suche
- ✅ `DELETE /seeds/:id` - Memory löschen
- ✅ `POST /seeds/generate-embeddings` - Embeddings nachziehen

**Vorteile:**
- ✅ Zusätzliche Features: Delete, Stats, Generate-Embeddings
- ✅ Lokale Verarbeitung (keine Cloud-Latenz)
- ✅ Keine API-Key erforderlich

## Agent Contexts (Session Persistence)

### OpenClaw Guide (Neutron)

```bash
# Context erstellen
./scripts/neutron-memory.sh context-create "my-agent" "episodic" '{"key":"value"}'

# Contexts auflisten
./scripts/neutron-memory.sh context-list "my-agent"

# Context abrufen
./scripts/neutron-memory.sh context-get abc-123
```

**Memory-Typen:**
- `episodic` - Konversationsverlauf, Entscheidungen, Aktionen
- `semantic` - Domänenwissen, Nutzerpräferenzen
- `procedural` - System-Prompts, Tool-Definitionen
- `working` - Aktueller Task-Status, Variablen

### Cortex

```bash
# Context erstellen
./cortex-cli context-create "my-agent" episodic '{}'

# Contexts auflisten
./cortex-cli context-list "my-agent"

# Context abrufen
./cortex-cli context-get <id>
```

**API-Endpunkte:**
- ✅ `POST /agent-contexts` - Context erstellen
- ✅ `GET /agent-contexts` - Contexts auflisten
- ✅ `GET /agent-contexts/:id` - Context abrufen

**Status:** ✅ Vollständig implementiert und Neutron-kompatibel

## API-Endpunkte Vergleich

| Endpoint | Neutron | Cortex | Status |
|----------|---------|--------|--------|
| `POST /seeds` | ✅ | ✅ | Identisch |
| `POST /seeds/query` | ✅ | ✅ | Identisch |
| `DELETE /seeds/:id` | ❓ | ✅ | Zusätzlich |
| `POST /agent-contexts` | ✅ | ✅ | Identisch |
| `GET /agent-contexts` | ✅ | ✅ | Identisch |
| `GET /agent-contexts/:id` | ✅ | ✅ | Identisch |
| `GET /stats` | ❓ | ✅ | Zusätzlich |
| `POST /seeds/generate-embeddings` | ❓ | ✅ | Zusätzlich |

## Zusätzliche Cortex-Features

Cortex bietet Features, die im Neutron-Guide nicht erwähnt werden:

1. **Entities & Relations** (Knowledge Graph)
   ```bash
   ./cortex-cli entity-add carsten lieblingsfarbe blau
   ./cortex-cli relation-add carsten typescript programmiert
   ```

2. **Bundles** (Memory-Organisation)
   - Memories in logische Gruppen organisieren
   - Bundle-basierte Suche

3. **Export/Import**
   - Vollständige Daten-Migration
   - Backup/Restore

4. **Webhooks**
   - Event-Benachrichtigungen für Memory-Änderungen

5. **Analytics**
   - Dashboard-Daten über API

6. **Rate Limiting**
   - Token-Bucket-Algorithmus für API-Schutz

## Unterschiede im Detail

### 1. Installation

**Neutron:**
- ClawHub-basiert (`clawhub install`)
- Skill wird in `./skills/vanar-neutron-memory/` installiert
- Abhängig von externem Service

**Cortex:**
- Lokaler Server (Go-Binary)
- Skill in `./skills/cortex/` (geplant)
- Keine externen Abhängigkeiten

### 2. Credentials

**Neutron:**
- API-Key erforderlich
- Agent-ID erforderlich
- User-Identifier erforderlich
- Konfiguration via `.env` oder `~/.config/neutron/credentials.json`

**Cortex:**
- ❌ **Kein API-Key erforderlich** (lokal)
- Optional: `CORTEX_API_URL`, `CORTEX_APP_ID`, `CORTEX_USER_ID`
- Konfiguration via `.env` (optional)

### 3. Shell-Skripte

**Neutron:**
- `./scripts/neutron-memory.sh` für alle Operationen
- Wrapper um API-Calls

**Cortex:**
- `./cortex-cli` (Go-Binary) für alle Operationen
- Keine Shell-Skript-Abhängigkeiten
- Direkte API-Calls möglich

### 4. Hooks

**Neutron:**
- Auto-Recall/Auto-Capture über Skill-Hooks
- Konfigurierbar via Env-Variablen

**Cortex:**
- ✅ Implementiert
- `skills/cortex/hooks.sh` mit `recall` und `capture` Subcommands
- Unterstützt JSON und Plain-Text Input
- Konfigurierbar via Env-Variablen

## Migration von Neutron zu Cortex

### Schritt 1: API-Client ändern

**Vorher (Neutron):**
```typescript
import { NeutronClient } from '@vanar/neutron-sdk';

const client = new NeutronClient({
    apiKey: 'nk_...',
    baseUrl: 'https://api-neutron.vanarchain.com'
});
```

**Nachher (Cortex):**
```typescript
import { CortexClient } from '@openclaw/cortex-sdk';

const client = new CortexClient({
    baseUrl: 'http://localhost:9123' // Lokaler Server
    // Kein API-Key erforderlich!
});
```

### Schritt 2: Code anpassen

**Minimale Änderungen:**
- Base-URL auf Cortex ändern (`http://localhost:9123`)
- Auth-Header entfernen (kein API-Key)
- API-Calls bleiben identisch (Neutron-kompatibel)

## Fazit

### ✅ Was Cortex bietet (wie Neutron)

- ✅ Seeds-API (Speichern, semantische Suche)
- ✅ Agent Contexts (Session Persistence)
- ✅ Multi-Tenant-Support
- ✅ Semantische Suche mit Embeddings
- ✅ Cross-Platform Continuity

### ✅ Was Cortex zusätzlich bietet

- ✅ Entities & Relations (Knowledge Graph)
- ✅ Bundles (Memory-Organisation)
- ✅ Export/Import
- ✅ Webhooks
- ✅ Analytics
- ✅ Rate Limiting
- ✅ Lokale Verarbeitung (keine Cloud-Latenz)
- ✅ **Kein API-Key erforderlich**

### ✅ Vollständig implementiert

- ✅ OpenClaw Skill-Integration (hooks.sh verfügbar)
- ✅ Auto-Recall/Auto-Capture Hooks (implementiert)
- ✅ Shell-Skripte für Hooks (hooks.sh)

### 🎯 Hauptvorteile von Cortex

1. **Lokal & Privat**: Keine Cloud-Abhängigkeit, Daten bleiben lokal
2. **Kostenlos**: Keine API-Kosten, selbst gehostet
3. **Kein API-Key**: Einfache Installation ohne Credentials
4. **Schneller**: Lokale Verarbeitung, keine Netzwerk-Latenz
5. **Erweiterbar**: Zusätzliche Features (Entities, Relations, etc.)

## Nächste Schritte

1. ✅ Server läuft lokal
2. ✅ CLI-Tool verfügbar
3. ✅ API-Endpunkte implementiert
4. ⏳ OpenClaw Skill-Integration (geplant)
5. ⏳ Auto-Recall/Auto-Capture Hooks (geplant)
