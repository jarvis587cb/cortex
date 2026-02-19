# Neutron Memory API Analyse

**Quelle:** https://openclaw.vanarchain.com/  
**Datum:** 2026-02-19

## Produkt-Übersicht

**Neutron Memory API** ist eine **Cloud-basierte Memory-Plattform** für OpenClaw-Agenten, entwickelt von Vanar Chain. Sie bietet persistente, durchsuchbare Memory-Funktionalität mit semantischer Suche und Multi-Modal-Embeddings.

## Kern-Features

### 1. Performance
- ✅ **Sub-200ms Search** – Blitzschnelle semantische Suche
- ✅ **PostgreSQL + pgvector** – Professionelle Datenbank-Infrastruktur
- ✅ **Multimodal Embeddings** – 1024-dimensionale Jina v4 Embeddings
- ✅ **100+ Sprachen** – Native Multilingual-Unterstützung

### 2. Funktionalität
- ✅ **Multi-Tenant** – Eingebaute Unterstützung für externe User
- ✅ **Bundles** – Organisation von Wissen in logische Gruppen
- ✅ **RESTful API** – Saubere REST-API mit TypeScript SDK
- ✅ **Semantic Search** – Suche über Text, Bilder und Dokumente

### 3. Use Cases
- Personal AI Assistants
- RAG Applications
- Customer Support Bots
- Knowledge Management
- OpenClaw Agents
- Multi-User Apps

## API-Struktur

### Endpunkte (aus Beispiel-Code)

**Base URL:** `https://api-neutron.vanarchain.com`

#### Memory speichern
```
POST /seeds?appId={AGENT_ID}&externalUserId={AGENT_IDENTIFIER}
Authorization: Bearer nk_...
Content-Type: application/json

{
  "content": "...",
  "metadata": { "userId": "...", "type": "..." }
}
```

#### Memory-Suche
```
POST /seeds/query?appId={AGENT_ID}&externalUserId={AGENT_IDENTIFIER}
Authorization: Bearer nk_...
Content-Type: application/json

{
  "query": "...",
  "limit": 5
}
```

## Vergleich: Neutron vs. Cortex

### Gemeinsamkeiten ✅

| Feature | Neutron | Cortex |
|---------|---------|--------|
| **Seeds-API** | ✅ `/seeds`, `/seeds/query` | ✅ `/seeds`, `/seeds/query` |
| **Multi-Tenant** | ✅ `appId` + `externalUserId` | ✅ `appId` + `externalUserId` |
| **REST-API** | ✅ RESTful | ✅ RESTful |
| **Memory-Speicherung** | ✅ Persistent | ✅ Persistent (SQLite) |
| **Query-Parameter** | ✅ Query-String | ✅ Query-String + Body |
| **Metadata-Support** | ✅ JSON Metadata | ✅ JSON Metadata |

### Unterschiede ⚠️

| Aspekt | Neutron | Cortex |
|--------|---------|--------|
| **Deployment** | ☁️ Cloud (SaaS) | 🏠 Lokal (Self-hosted) |
| **Datenbank** | PostgreSQL + pgvector | SQLite (pure-Go) |
| **Embeddings** | ✅ Jina v4 (1024-dim) | ❌ Keine (Textsuche mit LIKE) |
| **Semantische Suche** | ✅ Cosine-Similarity | ⚠️ Textsuche (Heuristik) |
| **Multimodal** | ✅ Text + Bilder + Docs | ❌ Nur Text |
| **Performance** | Sub-200ms | Abhängig von Datenmenge |
| **Skalierung** | ✅ Hochskalierbar | ⚠️ Single-Instance (SQLite) |
| **Kosten** | 💰 Pay-per-use | ✅ Kostenlos (Self-hosted) |
| **Authentifizierung** | ✅ Bearer Token (nk_...) | ✅ Optional API-Key |
| **Bundles** | ✅ Unterstützt | ❌ Nicht implementiert |
| **Sprachen** | ✅ 100+ (Jina v4) | ⚠️ Basis-Unterstützung |

## API-Kompatibilität

### Request-Format Vergleich

**Neutron:**
```javascript
// Query-String Parameter
POST /seeds?appId=xxx&externalUserId=yyy
Body: { content, metadata }
```

**Cortex:**
```javascript
// Body-Parameter (kompatibel)
POST /seeds
Body: { appId, externalUserId, content, metadata }
```

**Unterschied:** Neutron nutzt Query-Parameter für Tenant-IDs, Cortex nutzt Body-Parameter. Beide sind kompatibel, wenn man die Parameter entsprechend mappt.

### Response-Format

**Beide APIs** verwenden ähnliche Response-Strukturen:
- `id` – Memory-ID
- `content` – Textinhalt
- `metadata` – JSON-Metadaten
- `created_at` – Zeitstempel
- `similarity` – Ähnlichkeits-Score (bei Queries)

## Vorteile von Neutron

1. **Professionelle Infrastruktur**
   - Cloud-basiert, keine Wartung nötig
   - Hochskalierbar
   - Professionelles Monitoring

2. **Semantische Suche**
   - Echte Embeddings (Jina v4)
   - Multimodal (Text, Bilder, Dokumente)
   - Präzise Suchergebnisse

3. **Performance**
   - Sub-200ms Response-Zeit
   - Optimiert für große Datenmengen
   - CDN-Integration möglich

4. **Features**
   - Bundles für Organisation
   - Multilingual (100+ Sprachen)
   - TypeScript SDK

## Vorteile von Cortex

1. **Lokale Kontrolle**
   - Self-hosted, keine Cloud-Abhängigkeit
   - Daten bleiben lokal
   - Keine API-Kosten

2. **Einfachheit**
   - Leichtgewichtig (16MB Binary)
   - Keine externen Dependencies
   - Einfache Installation

3. **Privacy**
   - Daten bleiben auf eigenem Server
   - Keine Datenübertragung ins Internet
   - Vollständige Kontrolle

4. **Kosten**
   - Komplett kostenlos
   - Keine API-Limits
   - Keine Subscription-Gebühren

## Migrations-Pfad

### Von Neutron zu Cortex

**Vorteile:**
- ✅ Gleiche API-Struktur (Seeds-API)
- ✅ Gleiche Request/Response-Formate
- ✅ Multi-Tenant-Support vorhanden
- ✅ Einfache Migration möglich

**Herausforderungen:**
- ⚠️ Keine semantische Suche (nur Textsuche)
- ⚠️ Keine Embeddings
- ⚠️ Performance bei großen Datenmengen

### Von Cortex zu Neutron

**Vorteile:**
- ✅ Upgrade auf semantische Suche
- ✅ Multimodal-Support
- ✅ Bessere Performance
- ✅ Professionelle Infrastruktur

**Herausforderungen:**
- ⚠️ Cloud-Abhängigkeit
- ⚠️ Kosten
- ⚠️ Daten-Migration nötig

## Empfehlungen

### Wann Neutron verwenden?

✅ **Empfohlen für:**
- Produktions-Umgebungen mit hohem Traffic
- Anwendungen mit semantischer Suche
- Multimodal-Content (Bilder, Dokumente)
- Multi-Language-Anwendungen
- Teams ohne DevOps-Ressourcen

### Wann Cortex verwenden?

✅ **Empfohlen für:**
- Entwicklung & Testing
- Lokale/Private Anwendungen
- Privacy-kritische Anwendungen
- Kleine bis mittlere Datenmengen
- Budget-bewusste Projekte

## Hybrid-Ansatz

**Möglichkeit:** Beide Systeme parallel nutzen:

1. **Cortex** für Development/Testing
2. **Neutron** für Production
3. **Gleiche API-Struktur** ermöglicht einfaches Umschalten

**Vorteile:**
- Kostenoptimierung (Development kostenlos)
- Flexibilität (lokale Tests)
- Einfache Migration (gleiche API)

## Fazit

**Neutron Memory API** ist eine **professionelle, Cloud-basierte Lösung** mit semantischer Suche und Multimodal-Support. **Cortex** ist eine **lokale, leichtgewichtige Alternative** mit Neutron-kompatibler API.

**Beide Systeme ergänzen sich:**
- **Neutron** für Production mit hohen Anforderungen
- **Cortex** für Development, Testing und Privacy-kritische Anwendungen

**Dein Cortex-Projekt** bietet bereits eine **solide Basis** mit Neutron-Kompatibilität und kann als:
- ✅ Lokale Alternative zu Neutron
- ✅ Development/Testing-Umgebung
- ✅ Privacy-fokussierte Lösung

verwendet werden.

## Nächste Schritte

### Für Cortex-Entwicklung:

1. **Embeddings hinzufügen** (optional)
   - Jina v4 Integration
   - Semantische Suche implementieren
   - Performance optimieren

2. **Bundles-Feature** (optional)
   - Organisation von Memories
   - Gruppierung nach Projekten/Topics

3. **Migration-Tools** (optional)
   - Export/Import von Neutron-Daten
   - Synchronisation zwischen Systemen

4. **Dokumentation erweitern**
   - Neutron-Kompatibilität hervorheben
   - Migrations-Guide erstellen
   - Vergleichs-Tabelle aktualisieren
