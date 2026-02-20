---
name: cortex
description: |-
  Lokales Gedächtnis für OpenClaw. Speichert Memories, Fakten, Relationen in SQLite.
  Nutze für: (1) Memories speichern/suchen, (2) Entities mit Fakten, (3) Relations (Knowledge Graph),
  (4) Session-Context.
  
  Vergleich mit OpenClaw Neutron Guide: Siehe [docs/VERGLEICH_OPENCLAW_GUIDE.md](../../docs/VERGLEICH_OPENCLAW_GUIDE.md)
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
cortex-cli store "Carsten bevorzugt dunkles Theme" '{"typ":"persönlich","kategorie":"präferenz"}'  # Mit Metadata-Typ
cortex-cli store "Gateway restart um 14:30" '{"typ":"system","kategorie":"gateway"}'  # System-Event
cortex-cli query "Suchbegriff" 10              # Semantische Suche
cortex-cli query "Theme" 10 0.5 "" '{"typ":"persönlich"}'  # Suche mit Metadata-Filter
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

# API-Key-Verwaltung
cortex-cli api-key create [env_file]    # API-Key generieren und in .env speichern
cortex-cli api-key show [env_file]     # Aktuellen API-Key anzeigen (letzte 4 Zeichen)
cortex-cli api-key delete [env_file]  # API-Key aus .env entfernen

# Embeddings
cortex-cli generate-embeddings [batchSize]  # Embeddings für Memories nachziehen (Standard: 10, Max: 100)
```

**Hinweis:** Die API-Key-Funktion ist optional. Für lokale Installationen ist kein API-Key erforderlich. Die Funktion ist nützlich für Produktionsumgebungen oder wenn mehrere Clients auf denselben Server zugreifen.

**Beispiele:**
```bash
# API-Key erstellen (Standard: .env im Projekt-Root)
cortex-cli api-key create

# API-Key in spezifischer Datei erstellen
cortex-cli api-key create /path/to/.env

# Aktuellen Key anzeigen
cortex-cli api-key show

# Key löschen
cortex-cli api-key delete

# Embeddings für bestehende Memories generieren
cortex-cli generate-embeddings 50
```

## Metadata-Typen und Kategorien

Cortex unterstützt strukturierte Metadata-Typen, um Memories zu kategorisieren und zu filtern:

### Verfügbare Typen

- **`persönlich`**: Präferenzen, persönliche Informationen
  ```bash
  cortex-cli store "Carsten bevorzugt dunkles Theme" '{"typ":"persönlich","kategorie":"präferenz"}'
  ```

- **`system`**: Gateway-Checks, Cron-Logs, System-Events
  ```bash
  cortex-cli store "Gateway restart um 14:30" '{"typ":"system","kategorie":"gateway"}'
  ```

- **`bash`**: Wichtige Commands aus Bash-History
  ```bash
  cortex-cli store "docker-compose up -d" '{"typ":"bash","kategorie":"docker"}'
  ```

- **`decision`**: Wichtige Entscheidungen
  ```bash
  cortex-cli store "Migration zu PostgreSQL beschlossen" '{"typ":"decision","kategorie":"architektur"}'
  ```

### Suche mit Metadata-Filter

```bash
# Nur persönliche Memories suchen
cortex-cli query "Theme" 10 0.5 "" '{"typ":"persönlich"}'

# Nur System-Events suchen
cortex-cli query "Gateway" 10 0.5 "" '{"typ":"system"}'

# Nach Kategorie filtern
cortex-cli query "Docker" 10 0.5 "" '{"kategorie":"docker"}'

# Kombination von Filtern
cortex-cli query "Restart" 10 0.5 "" '{"typ":"system","kategorie":"gateway"}'
```

## Embeddings & Semantische Suche

Cortex unterstützt semantische Suche mit **vollständig lokalen Embeddings**. Beim Speichern von Memories werden automatisch Embeddings generiert, die für die semantische Suche verwendet werden.

### Embedding-Modi

Cortex bietet zwei Embedding-Methoden:

#### 1. **GTE-Small Modell** (Empfohlen für beste Qualität)

- ✅ **384-dimensionale Embeddings** – GTE-Small Modell (Alibaba DAMO Academy)
- ✅ **Hochwertige Semantik** – State-of-the-art Text-Embeddings
- ✅ **Vollständig lokal** – Keine externe API nötig
- ✅ **Keine API-Keys** – Funktioniert komplett offline
- ⚠️ **Modell-Download erforderlich** – ~70MB Modell-Datei

**Setup:**
```bash
# 1. Modell herunterladen und konvertieren
./scripts/download-gte-model.sh

# 2. In .env aktivieren
echo "CORTEX_EMBEDDING_MODEL_PATH=~/.openclaw/gte-small.gtemodel" >> .env

# 3. Server neu starten
systemctl --user restart cortex.service
```

#### 2. **Hash-basierter Service** (Standard, kein Download)

- ✅ **384-dimensionale Embeddings** – Lokale Hash-basierte Generierung
- ✅ **Sofort einsatzbereit** – Kein Download erforderlich
- ✅ **Vollständig offline** – Keine externe API nötig
- ✅ **Keine API-Keys** – Funktioniert ohne Konfiguration
- ✅ **Synonym-Erweiterung** – Begriffe wie Kaffee/Latte/Espresso werden verknüpft
- ⚠️ **Niedrigere Qualität** – Für einfache Anwendungen ausreichend

**Standard-Verhalten:** Wenn `CORTEX_EMBEDDING_MODEL_PATH` nicht gesetzt ist, wird automatisch der Hash-Service verwendet.

### Embeddings nachziehen

Wenn Memories ohne Embeddings gespeichert wurden (z.B. vor Aktivierung des GTE-Modells), können Embeddings nachträglich generiert werden:

```bash
# Embeddings für 10 Memories generieren (Standard)
cortex-cli generate-embeddings

# Embeddings für 50 Memories generieren
cortex-cli generate-embeddings 50

# Embeddings für maximal 100 Memories generieren
cortex-cli generate-embeddings 100
```

**Hinweis:** Der Befehl verarbeitet Memories in Batches. Bei großen Datenmengen kann es sinnvoll sein, den Befehl mehrfach auszuführen.

### Konfiguration

Embeddings werden über Umgebungsvariablen konfiguriert:

```bash
# GTE-Small Modell aktivieren (optional)
CORTEX_EMBEDDING_MODEL_PATH=~/.openclaw/gte-small.gtemodel

# Ohne diese Variable wird automatisch der Hash-basierte Service verwendet
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
| POST | /seeds/generate-embeddings | Embeddings nachziehen |
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
