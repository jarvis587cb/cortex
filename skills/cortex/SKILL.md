---
name: cortex
description: |-
  Lokales Gedächtnis für OpenClaw. Speichert Memories, Fakten, Relationen in SQLite.
  Nutze für: (1) Infos dauerhaft speichern, (2) Semantische Suche, (3) Session-Context.
---

# Cortex – Lokales Gedächtnis

## Schnellbefehle

```bash
# Basis-Operationen
./cortex-cli health                 # Health Check
./cortex-cli store "Text"           # Speichern
./cortex-cli query "Suchbegriff" 10  # Suchen
./cortex-cli delete <id>            # Löschen
./cortex-cli stats                  # Statistiken

# Context-Management
./cortex-cli context-create "agent" episodic '{"topic":"..."}'  # Context erstellen
./cortex-cli context-list [agentId]  # Contexts auflisten
./cortex-cli context-get <id>        # Context abrufen

# Erweiterte Funktionen
./cortex-cli generate-embeddings [batchSize]  # Embeddings nachziehen (Standard: 10)
./cortex-cli benchmark [count]                 # Performance-Benchmark (Standard: 20)
./cortex-cli api-key <create|delete|show>      # API-Key verwalten
```

## Workflows

**Speichern & Suchen:**
```bash
./cortex-cli store "Carsten mag Kaffee" '{"type":"fact"}'
./cortex-cli query "Kaffee" 5
```

**Session-Context:**
```bash
./cortex-cli context-create "agent" episodic '{"topic":"..."}'
./cortex-cli context-list "agent"
./cortex-cli context-get <id>
```
