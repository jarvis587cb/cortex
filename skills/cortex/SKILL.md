---
name: cortex
description: |-
  Lokales Gedächtnis für OpenClaw. Speichert Memories, Fakten, Relationen in SQLite.
  Nutze für: (1) Memories speichern/suchen, (2) Entities mit Fakten, (3) Relations (Knowledge Graph),
  (4) Session-Context.
---

# Cortex – Lokales Gedächtnis

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
