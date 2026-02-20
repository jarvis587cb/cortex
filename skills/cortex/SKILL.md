---
name: cortex
description: |-
  Lokales Gedächtnis für OpenClaw. Speichert Memories, Fakten, Relationen in SQLite.
  Nutze für: (1) Memories speichern/suchen, (2) Entities mit Fakten, (3) Relations (Knowledge Graph),
  (4) Session-Context.
  
  Vergleich mit OpenClaw Neutron Guide: Siehe [docs/VERGLEICH_OPENCLAW_GUIDE.md](../../docs/VERGLEICH_OPENCLAW_GUIDE.md)
---

# Cortex – Lokales Gedächtnis

> **Hinweis:** Cortex ist eine lokale Alternative zu Vanar Neutron. **Kein API-Key erforderlich!**  
> Vergleich mit dem [OpenClaw Integration Guide](https://openclaw.vanarchain.com/guide-openclaw): Siehe [docs/VERGLEICH_OPENCLAW_GUIDE.md](../../docs/VERGLEICH_OPENCLAW_GUIDE.md)

## Server

```bash
# Systemd-User-Konfiguration neu laden
systemctl --user daemon-reload

# Dienst beim Login aktivieren
systemctl --user enable cortex.service

# Dienst sofort starten
systemctl --user start cortex.service

# Status abfragen
systemctl --user status cortex.service

# Logs einsehen
journalctl --user -u cortex.service -f
```

## CLI (cortex-cli)

```bash
# Health
cortex-cli health                              # Health check

# Memories
cortex-cli store "Text" '[{"type":"fact"}]'  # Speichern
cortex-cli query "Suchbegriff" 10              # Semantische Suche
cortex-cli delete <id>                        # Löschen
cortex-cli stats                              # Stats

# Entities (Key-Value Fakten)
cortex-cli entity-add <entity> <key> <value>      # Fact hinzufügen
cortex-cli entity-get <entity>                    # Entity abrufen

# Relations (Knowledge Graph)
cortex-cli relation-add <from> <to> <type>       # Relation anlegen
cortex-cli relation-get <from>                   # Relations abrufen

# Context
cortex-cli context-create "agent" episodic '{}'
cortex-cli context-list "agent"
cortex-cli context-get <id>
```

## Hooks (Auto-Recall / Auto-Capture)

Cortex unterstützt automatisches Abrufen (Recall) und Speichern (Capture) von Memories über OpenClaw Skill-Hooks.

### Konfiguration

Hooks können über Umgebungsvariablen konfiguriert werden:

```bash
# Hooks aktivieren/deaktivieren
CORTEX_AUTO_RECALL=true      # Default: true
CORTEX_AUTO_CAPTURE=true     # Default: true

# API-Konfiguration
CORTEX_API_URL=http://localhost:9123
CORTEX_APP_ID=openclaw
CORTEX_USER_ID=default
CORTEX_API_KEY=              # Optional, nur für Produktion

# Recall-Parameter
CORTEX_RECALL_LIMIT=5        # Max Ergebnisse (Default: 5)
CORTEX_RECALL_THRESHOLD=0.5  # Ähnlichkeitsschwelle 0-1 (Default: 0.5)
```

### Recall-Hook (Vor AI-Interaktion)

Ruft relevante Memories ab, bevor der AI-Agent antwortet.

**Aufruf:**
```bash
echo '{"message": "user question"}' | hooks.sh recall
# oder
echo "user question" | hooks.sh recall
```

**Input-Format:**
- JSON: `{"message": "text", "appId": "...", "userId": "..."}`
- Plain-Text: Direkt als Query-Text

**Output-Format:**
```json
[
  {
    "id": 1,
    "content": "User prefers coffee",
    "similarity": 0.92,
    "metadata": {...}
  }
]
```

**Beispiel:**
```bash
# JSON-Input
echo '{"message": "Was weißt du über Kaffee?", "appId": "my-app", "userId": "user123"}' | hooks.sh recall

# Plain-Text-Input
echo "Was weißt du über Kaffee?" | hooks.sh recall
```

### Capture-Hook (Nach Konversation)

Speichert Konversationen automatisch nach jedem Austausch.

**Aufruf:**
```bash
echo '{"content": "User: Hello\nAI: Hi there!", "appId": "openclaw", "userId": "user123"}' | hooks.sh capture
```

**Input-Format:**
```json
{
  "content": "User: Hello\nAI: Hi there!",
  "appId": "openclaw",
  "userId": "user123",
  "metadata": {
    "platform": "discord",
    "channel": "#general"
  }
}
```

**Beispiel:**
```bash
# Konversation speichern
cat <<EOF | hooks.sh capture
{
  "content": "User: Was ist Cortex?\nAI: Cortex ist ein lokales Gedächtnis-System.",
  "appId": "openclaw",
  "userId": "user123",
  "metadata": {"platform": "discord"}
}
EOF
```

### Verwendung in OpenClaw

Die Hooks werden automatisch von OpenClaw aufgerufen:

1. **Vor AI-Interaktion:** `hooks.sh recall` wird mit User-Message aufgerufen
2. **Nach Konversation:** `hooks.sh capture` wird mit vollständiger Konversation aufgerufen

**Hooks deaktivieren:**
```bash
# In .env oder Umgebungsvariablen
CORTEX_AUTO_RECALL=false
CORTEX_AUTO_CAPTURE=false
```

## API-Endpunkte

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| POST | /seeds | Memory speichern |
| POST | /seeds/query | Semantische Suche |
| DELETE | /seeds/:id | Memory löschen |
| POST | /entities?entity=... | Fact hinzufügen |
| GET | /entities?name=... | Entity abrufen |
| POST | /relations | Relation anlegen |
| GET | /relations?from=... | Relations abrufen |
| GET | /stats | Übersicht |

## Beispiele

**Entity anlegen:**
```bash
cortex-cli entity-add carsten lieblingsfarbe blau
```

**Relation anlegen:**
```bash
cortex-cli relation-add carsten typescript programmiert
```

**Stats:**
```bash
cortex-cli stats
# {"memories":5,"entities":2,"relations":4}
```
