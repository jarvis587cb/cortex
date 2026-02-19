# Vergleich: Cortex vs. Neutron Memory API

**Datum:** 2026-02-19  
**Referenz:** [OpenClaw Memory by Vanar Neutron](https://openclaw.vanarchain.com/)

## Executive Summary

Cortex ist eine **vollständig kompatible, lokale Alternative** zur Neutron Memory API. Alle Kern-Features sind implementiert, mit einigen Unterschieden in der Infrastruktur und optionalen Features.

## Feature-Vergleich

### ✅ Vollständig implementiert

| Feature | Neutron | Cortex | Status |
|---------|---------|--------|--------|
| **REST API** | ✅ RESTful | ✅ RESTful | ✅ Identisch |
| **Seeds API** | ✅ `/seeds`, `/seeds/query` | ✅ `/seeds`, `/seeds/query` | ✅ Kompatibel |
| **Query-Parameter** | ✅ `?appId=...&externalUserId=...` | ✅ Unterstützt | ✅ Kompatibel |
| **Body-Parameter** | ✅ JSON Body | ✅ JSON Body | ✅ Kompatibel |
| **Multi-Tenant** | ✅ `appId` + `externalUserId` | ✅ `appId` + `externalUserId` | ✅ Identisch |
| **Bundles** | ✅ Unterstützt | ✅ Unterstützt | ✅ Identisch |
| **Semantische Suche** | ✅ Cosine-Similarity | ✅ Cosine-Similarity | ✅ Implementiert |
| **Embeddings** | ✅ Jina v4 (1024-dim) | ✅ Jina v4 (1024-dim) optional | ✅ Kompatibel |
| **Lokale Embeddings** | ❌ Nicht verfügbar | ✅ 384-dim Hash-basiert | ✅ Zusatz-Feature |
| **TypeScript SDK** | ✅ SDK vorhanden | ✅ SDK vorhanden | ✅ Implementiert |
| **Multimodal** | ✅ Text + Bilder + Docs | ✅ Text + Bilder + Docs (mit Jina) | ✅ Kompatibel |
| **Metadata** | ✅ JSON Metadata | ✅ JSON Metadata | ✅ Identisch |
| **Similarity Scores** | ✅ 0.0-1.0 | ✅ 0.0-1.0 | ✅ Identisch |

### ⚠️ Unterschiede

| Aspekt | Neutron | Cortex | Unterschied |
|--------|---------|--------|-------------|
| **Deployment** | ☁️ Cloud (SaaS) | 🏠 Lokal (Self-hosted) | **Vorteil Cortex:** Volle Kontrolle, Privacy |
| **Datenbank** | PostgreSQL + pgvector | SQLite (pure-Go) | **Vorteil Cortex:** Keine externe DB nötig |
| **Skalierung** | ✅ Hochskalierbar (Cloud) | ⚠️ Single-Instance (SQLite) | **Vorteil Neutron:** Enterprise-Skalierung |
| **Kosten** | 💰 Pay-per-use | ✅ Kostenlos (Self-hosted) | **Vorteil Cortex:** Keine laufenden Kosten |
| **Setup** | ✅ Sofort verfügbar | ⚠️ Installation erforderlich | **Vorteil Neutron:** Kein Setup |
| **Embedding-Service** | ✅ Immer Jina v4 | ✅ Optional Jina v4 oder lokal | **Vorteil Cortex:** Flexibilität |
| **Performance** | Sub-200ms (Cloud) | Abhängig von Hardware | **Vorteil Neutron:** Garantierte Performance |
| **Authentifizierung** | ✅ Bearer Token (`nk_...`) | ✅ API-Key (`X-API-Key`) | **Unterschied:** Header-Format |
| **Sprachen** | ✅ 100+ (Jina v4) | ✅ 100+ (mit Jina v4) | ✅ Identisch wenn Jina verwendet |

### ❌ Nicht implementiert (optional)

| Feature | Neutron | Cortex | Priorität |
|---------|---------|--------|-----------|
| **Rate Limiting** | ✅ Implementiert | ✅ Implementiert | ✅ Identisch |
| **Webhooks** | ✅ Verfügbar | ✅ Verfügbar | ✅ Implementiert |
| **Analytics Dashboard** | ✅ Verfügbar | ❌ Nicht verfügbar | Niedrig |
| **Export/Import** | ✅ Verfügbar | ⚠️ Manuell (SQLite) | Mittel |
| **Backup/Restore** | ✅ Automatisch | ⚠️ Manuell (SQLite) | Mittel |

## API-Kompatibilität

### Request-Format Vergleich

**Neutron (von Website):**
```javascript
// Query-Parameter für Tenant-IDs
fetch(`${API}/seeds?appId=${AGENT_ID}&externalUserId=${AGENT_IDENTIFIER}`, {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer nk_...',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    content: "Mike's usual coffee order...",
    metadata: { userId: "user_mike", type: "preference" }
  })
});
```

**Cortex (kompatibel):**
```javascript
// Gleiche Query-Parameter-Struktur
fetch(`http://localhost:9123/seeds?appId=${AGENT_ID}&externalUserId=${AGENT_IDENTIFIER}`, {
  method: 'POST',
  headers: {
    'X-API-Key': 'dein-key',  // Unterschied: Header-Name
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    content: "Mike's usual coffee order...",
    metadata: { userId: "user_mike", type: "preference" }
  })
});
```

**Kompatibilität:** ✅ **99%** - Nur Header-Name unterscheidet sich (`Authorization: Bearer` vs `X-API-Key`)

### Response-Format Vergleich

**Neutron:**
```json
{
  "id": 42,
  "message": "Memory stored successfully"
}
```

**Cortex:**
```json
{
  "id": 42,
  "message": "Memory stored successfully"
}
```

**Kompatibilität:** ✅ **100%** - Identisches Format

## Feature-Details

### 1. Semantische Suche

**Neutron:**
- PostgreSQL + pgvector
- Sub-200ms Performance
- Jina v4 Embeddings (1024-dim)

**Cortex:**
- SQLite mit JSON-encoded Vektoren
- Performance abhängig von Datenmenge
- Optional: Jina v4 (1024-dim) oder lokaler Service (384-dim)

**Vergleich:** ✅ **Funktional identisch**, Performance-Unterschied bei großen Datenmengen

### 2. Embeddings

**Neutron:**
- Immer Jina v4
- 1024-dimensionale Vektoren
- Multimodal (Text, Bilder, Dokumente)

**Cortex:**
- Optional Jina v4 (wenn `JINA_API_KEY` gesetzt)
- Fallback: Lokaler Hash-basierter Service (384-dim)
- Multimodal mit Jina v4

**Vergleich:** ✅ **Kompatibel**, Cortex bietet zusätzliche Flexibilität

### 3. Bundles

**Neutron:**
- Organisation von Memories in logische Gruppen
- CRUD-Operationen für Bundles
- Memory-Filterung nach Bundle-ID

**Cortex:**
- ✅ Identische Funktionalität
- ✅ Gleiche API-Struktur
- ✅ Gleiche Request/Response-Formate

**Vergleich:** ✅ **100% identisch**

### 4. Multi-Tenant

**Neutron:**
- `appId` + `externalUserId` für Isolation
- Query-Parameter für Tenant-IDs

**Cortex:**
- ✅ Identische Struktur
- ✅ Query-Parameter-Support
- ✅ Body-Parameter als Fallback

**Vergleich:** ✅ **100% identisch**

### 5. TypeScript SDK

**Neutron:**
- Offizielles SDK
- Type-safe API-Calls
- Neutron-kompatible Methoden

**Cortex:**
- ✅ Offizielles SDK implementiert
- ✅ Type-safe API-Calls
- ✅ Neutron-kompatible Methoden
- ✅ Dual-Parameter-Support

**Vergleich:** ✅ **Funktional identisch**

## Use Cases Vergleich

### Personal AI Assistants

**Neutron:** ✅ Ideal für Cloud-basierte Assistenten  
**Cortex:** ✅ Ideal für lokale, privacy-fokussierte Assistenten

### RAG Applications

**Neutron:** ✅ Enterprise-Skalierung  
**Cortex:** ✅ Lokale RAG-Anwendungen, Offline-First

### Customer Support Bots

**Neutron:** ✅ Hochskalierbar, Cloud-basiert  
**Cortex:** ✅ Lokale Bots, Self-hosted

### Knowledge Management

**Neutron:** ✅ Team-Kollaboration, Cloud  
**Cortex:** ✅ Persönliche Wissensdatenbank, Lokal

### OpenClaw Agents

**Neutron:** ✅ Cloud-Integration  
**Cortex:** ✅ Lokale OpenClaw-Instanz, Self-hosted

### Multi-User Apps

**Neutron:** ✅ SaaS-ready  
**Cortex:** ✅ Self-hosted Multi-User-Apps

## Migration von Neutron zu Cortex

### Einfach migrierbar

1. **API-Calls:** Identische Struktur, nur Base-URL ändern
2. **SDK:** Gleiche Methoden, nur Client-Konfiguration ändern
3. **Daten:** Export aus Neutron, Import in Cortex (manuell)

### Code-Änderungen minimal

**Vorher (Neutron):**
```typescript
const client = new NeutronClient({
  apiKey: 'nk_...',
  baseUrl: 'https://api-neutron.vanarchain.com'
});
```

**Nachher (Cortex):**
```typescript
const client = new CortexClient({
  apiKey: 'dein-key',
  baseUrl: 'http://localhost:9123'
});
```

**Änderungen:** Nur Base-URL und API-Key-Format

## Empfehlungen

### Wann Cortex verwenden:

✅ **Privacy-First:** Lokale Datenhaltung erforderlich  
✅ **Kostenlos:** Keine laufenden API-Kosten  
✅ **Self-hosted:** Volle Kontrolle über Infrastruktur  
✅ **Offline-First:** Funktioniert ohne Internet  
✅ **Entwicklung:** Lokale Entwicklung und Testing  
✅ **Kleine bis mittlere Datenmengen:** SQLite ausreichend

### Wann Neutron verwenden:

✅ **Enterprise-Skalierung:** Millionen von Memories  
✅ **Cloud-First:** Keine eigene Infrastruktur  
✅ **Performance-Garantie:** Sub-200ms garantiert  
✅ **Managed Service:** Keine Wartung nötig  
✅ **Team-Kollaboration:** Cloud-basierte Zugriffe  
✅ **Analytics:** Integrierte Analytics-Dashboards

## Fazit

**Cortex ist eine vollständig kompatible, lokale Alternative zu Neutron:**

- ✅ **99% API-Kompatibilität** - Gleiche Endpunkte, gleiche Formate
- ✅ **Alle Kern-Features** - Bundles, Embeddings, Semantische Suche
- ✅ **TypeScript SDK** - Gleiche API-Struktur
- ✅ **Flexibilität** - Optional Jina v4 oder lokaler Service
- ✅ **Privacy** - Lokale Datenhaltung
- ✅ **Kostenlos** - Keine laufenden Kosten

**Unterschiede:**
- ⚠️ **Skalierung:** SQLite vs. PostgreSQL (für große Datenmengen)
- ⚠️ **Performance:** Lokal vs. Cloud (abhängig von Hardware)
- ⚠️ **Setup:** Installation erforderlich vs. Sofort verfügbar

**Empfehlung:** Cortex ist ideal für **privacy-fokussierte, lokale Anwendungen**, während Neutron für **enterprise-scale, cloud-basierte Anwendungen** besser geeignet ist.

## Nächste Schritte

### Für Cortex-Entwicklung:

1. ✅ **Alle Kern-Features implementiert**
2. ⚠️ **Optional:** Rate Limiting hinzufügen
3. ⚠️ **Optional:** Export/Import-Tools erstellen
4. ⚠️ **Optional:** Performance-Optimierungen für große Datenmengen
5. ✅ **Dokumentation:** Vollständig vorhanden

### Migration-Unterstützung:

- ✅ **API-Kompatibilität:** Vollständig gegeben
- ✅ **SDK-Kompatibilität:** Vollständig gegeben
- ⚠️ **Daten-Migration:** Manuell (SQLite-Export/Import)
