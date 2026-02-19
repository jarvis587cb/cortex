# Cortex Memory Skill (OpenClaw)

Lokales Memory-Backend für OpenClaw – kompatibel mit dem [OpenClaw Integration Guide](https://openclaw.vanarchain.com/guide-openclaw) (Neutron-Interface). Nutzt die Cortex-API statt Neutron (kein API-Key, alles lokal).

## Was dieses Skill macht

- Stellt die gleichen **Script-Befehle** wie der Neutron-Guide bereit: `test`, `save`, `search`, `context-create`, `context-list`, `context-get`.
- Kann mit **Hooks** (Auto-Recall, Auto-Capture) integriert werden, sofern OpenClaw diese für das Skill aufruft (siehe unten).
- Verwendet **Cortex** als Backend (Seeds + Agent Contexts), kein Neutron-API-Key nötig.

## Installation

Cortex-Server muss laufen (siehe [skills/cortex/SKILL.md](../cortex/SKILL.md)).

### Option A: Im Cortex-Projekt (Script direkt)

```bash
# Aus dem Cortex-Projekt
./scripts/cortex-memory.sh test
./scripts/cortex-memory.sh save "User prefers oat milk lattes" "{}"
./scripts/cortex-memory.sh search "coffee preferences" 10 0.5
```

### Option B: In einem OpenClaw-Projekt

Skill-Ordner in dein OpenClaw-Projekt kopieren oder verlinken, z. B.:

```bash
# Beispiel: OpenClaw-Projekt
mkdir -p skills/cortex-memory
cp -r /path/to/cortex/skills/cortex-memory/* skills/cortex-memory/
# Script aus Cortex-Projekt aufrufbar machen oder in OpenClaw scripts/ kopieren
cp /path/to/cortex/scripts/cortex-memory.sh scripts/
chmod +x scripts/cortex-memory.sh
```

## Konfiguration

In deiner OpenClaw `.env` oder Umgebung:

```bash
# Cortex-API (statt Neutron)
CORTEX_API_URL=http://localhost:9123
CORTEX_APP_ID=openclaw
CORTEX_USER_ID=default
```

Für Abwärtskompatibilität mit dem Neutron-Guide-Namen:

```bash
NEUTRON_API_URL=http://localhost:9123
NEUTRON_AGENT_ID=openclaw
YOUR_AGENT_IDENTIFIER=default
```

Das Script `cortex-memory.sh` liest zuerst `CORTEX_*`, fallback auf `NEUTRON_*` / `YOUR_AGENT_IDENTIFIER`.

## Test

```bash
./scripts/cortex-memory.sh test
```

Prüft, ob die Cortex-API unter `CORTEX_API_URL` erreichbar ist (GET /health).

## Seeds (Memory Storage & Search)

### Memory speichern

```bash
./scripts/cortex-memory.sh save "User prefers oat milk lattes from Blue Bottle every weekday morning" "{}"
./scripts/cortex-memory.sh save "User coffee preference" '{"type":"preference"}'
```

### Semantische Suche

```bash
./scripts/cortex-memory.sh search "what do I know about coffee" 10 0.5
```

- `query` (erforderlich)
- `limit` (optional, Standard 30, 1–100)
- `threshold` (optional, 0–1, Standard 0.5)

## Agent Contexts (Session Persistence)

### Context erstellen

```bash
./scripts/cortex-memory.sh context-create "my-agent" "episodic" '{"conversation":[],"lastTopic":"coffee"}'
```

Memory-Typen: `episodic`, `semantic`, `procedural`, `working`.

### Contexts auflisten

```bash
./scripts/cortex-memory.sh context-list "my-agent"
```

### Einzelnen Context abrufen

```bash
./scripts/cortex-memory.sh context-get <id>
```

## Hooks (Auto-Recall / Auto-Capture)

Umsetzung wie im [OpenClaw-Guide](https://openclaw.vanarchain.com/guide-openclaw):

- **Auto-Recall:** Vor jeder AI-Interaktion relevante Vergangenheit abrufen.
- **Auto-Capture:** Nach jedem Austausch Konversation speichern.

### Aufruf

**Ein Einstiegspunkt (empfohlen für OpenClaw):**

```bash
# Vor Interaktion (Recall)
./skills/cortex-memory/hooks.sh recall "letzte User-Nachricht oder Thema"

# Nach Austausch (Capture)
./skills/cortex-memory/hooks.sh capture "Zusammenfassung oder Rohinhalt des Austauschs"
```

**Oder direkt das Script:**

```bash
./scripts/cortex-memory.sh recall "query" [limit] [threshold]
./scripts/cortex-memory.sh capture "content" [metadata_json]
```

`hooks.sh` ruft intern `cortex-memory.sh` auf. Wenn das Script woanders liegt, setze `CORTEX_MEMORY_SCRIPT` auf den vollen Pfad zu `cortex-memory.sh`.

### Umgebungsvariablen

- **VANAR_AUTO_RECALL** (default: `true`): Wenn `false` oder `0`, macht `recall` nichts (Exit 0).
- **VANAR_AUTO_CAPTURE** (default: `true`): Wenn `false` oder `0`, macht `capture` nichts (Exit 0).

So kann OpenClaw die Hooks konfigurierbar ein- oder ausschalten. Die Anbindung an OpenClaw-Events („vor Interaktion“ / „nach Austausch“) erfolgt in der OpenClaw- bzw. Skill-Integration (z. B. Skripte oder Hooks, die `hooks.sh recall` bzw. `hooks.sh capture` aufrufen).

## API-Endpunkte (Referenz)

| Method | Endpoint              | Beschreibung        |
|--------|------------------------|---------------------|
| POST   | /seeds                 | Memory speichern    |
| POST   | /seeds/query           | Semantische Suche   |
| POST   | /agent-contexts        | Agent Context anlegen |
| GET    | /agent-contexts        | Contexts auflisten  |
| GET    | /agent-contexts/{id}   | Ein Context abrufen |

Alle Endpunkte nutzen `appId` und `externalUserId` (Query oder Body). Kein API-Key nötig (Cortex läuft lokal).

## Siehe auch

- [CORTEX_NEUTRON_ALTERNATIVE.md](../../CORTEX_NEUTRON_ALTERNATIVE.md) – Vergleich mit Neutron und OpenClaw-Guide
- [API.md](../../API.md) – Vollständige Cortex-API
- [skills/cortex/SKILL.md](../cortex/SKILL.md) – Cortex-Server installieren und starten
