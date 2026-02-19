# Kryptographische Verifizierung: Evaluierung

**Datum:** 2026-02-19  
**Referenz:** Neutron-Artikel erwähnt "cryptographically verifiable knowledge units"

## Executive Summary

Cortex implementiert **kryptographische Verifizierung für Webhooks** (HMAC-SHA256), aber **nicht explizit für Seeds/Memories**. Für lokale Self-hosted Installationen ist dies ausreichend, da SQLite implizite Datenintegrität bietet. Für verteilte Szenarien oder Audit-Anforderungen könnte explizite Seed-Signierung sinnvoll sein.

## Aktueller Stand

### ✅ Webhooks: HMAC-SHA256 Signaturen

Cortex signiert Webhook-Payloads mit HMAC-SHA256:

```go
// internal/webhooks/webhooks.go
func (w *WebhookService) SignPayload(payload []byte) string {
    mac := hmac.New(sha256.New, []byte(w.secret))
    mac.Write(payload)
    return hex.EncodeToString(mac.Sum(nil))
}
```

**Verwendung:**
- Webhook-Payloads enthalten `X-Cortex-Signature` Header
- Empfänger können Signatur verifizieren
- Schutz vor Manipulation bei Übertragung

**Code-Beispiel:**
```go
// Webhook wird mit HMAC-SHA256 signiert
signature := webhookService.SignPayload(payload)
headers := map[string]string{
    "X-Cortex-Signature": signature,
    "Content-Type": "application/json",
}
// Empfänger kann Signatur verifizieren
```

### ⚠️ Seeds/Memories: Keine explizite Signatur

Aktuell haben Memories keine kryptographische Signatur:

```go
// internal/models/models.go
type Memory struct {
    ID            int64  `gorm:"primaryKey"`
    Content       string `gorm:"not null"`
    Embedding     string // JSON-encoded vector
    AppID         string `gorm:"index:idx_tenant"`
    ExternalUserID string `gorm:"index:idx_tenant"`
    // Kein Signatur-Feld
}
```

## Evaluierung: Ist Seed-Signierung nötig?

### Option A: SQLite-Integrität (aktuell) ✅

**Beschreibung:**
- SQLite bietet implizite Datenintegrität durch Checksums
- WAC (Write-Ahead Logging) für Konsistenz
- Page-Level Checksums für Fehlererkennung

**Vorteile:**
- ✅ Keine zusätzliche Komplexität
- ✅ Automatische Integritätsprüfung
- ✅ Keine Secret-Verwaltung nötig
- ✅ Ausreichend für lokale Installationen

**Nachteile:**
- ❌ Keine explizite Authentifizierung (wer hat Memory erstellt?)
- ❌ Keine Audit-Trail für Signatur-Änderungen
- ❌ Nicht geeignet für verteilte Szenarien

**Empfehlung:** ✅ **Ausreichend für lokale Self-hosted Installationen**

### Option B: HMAC-Signaturen für Seeds

**Beschreibung:**
- Ähnlich wie Webhooks: Content-Hash mit Secret
- Signatur wird beim Erstellen gespeichert
- Verifizierung bei Abfrage möglich

**Implementierung:**
```go
type Memory struct {
    // ... bestehende Felder
    Signature string `gorm:"index"` // HMAC-SHA256 Signatur
}

func SignMemory(mem *Memory, secret string) {
    content := fmt.Sprintf("%s|%s|%s", mem.Content, mem.AppID, mem.ExternalUserID)
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(content))
    mem.Signature = hex.EncodeToString(mac.Sum(nil))
}

func VerifyMemory(mem *Memory, secret string) bool {
    expected := SignMemory(mem, secret)
    return hmac.Equal([]byte(mem.Signature), []byte(expected))
}
```

**Vorteile:**
- ✅ Explizite Verifizierung möglich
- ✅ Schutz vor Manipulation
- ✅ Audit-Trail für Signatur-Änderungen
- ✅ Konsistent mit Webhook-Implementierung

**Nachteile:**
- ❌ Zusätzliche Komplexität
- ❌ Secret-Management erforderlich
- ❌ Performance-Overhead (minimal)
- ❌ Migration bestehender Memories nötig

**Empfehlung:** ⚠️ **Sinnvoll für verteilte Szenarien oder Audit-Anforderungen**

### Option C: Content-Hash speichern

**Beschreibung:**
- SHA-256 Hash des Contents als Feld
- Keine Secret-Verwaltung nötig
- Nur Integritätsprüfung, keine Authentifizierung

**Implementierung:**
```go
type Memory struct {
    // ... bestehende Felder
    ContentHash string `gorm:"index"` // SHA-256 Hash
}

func HashMemory(mem *Memory) {
    h := sha256.New()
    h.Write([]byte(mem.Content))
    mem.ContentHash = hex.EncodeToString(h.Sum(nil))
}

func VerifyMemoryHash(mem *Memory) bool {
    expected := HashMemory(mem)
    return mem.ContentHash == expected
}
```

**Vorteile:**
- ✅ Integritätsprüfung ohne Secret
- ✅ Einfache Implementierung
- ✅ Keine Secret-Verwaltung

**Nachteile:**
- ❌ Keine Authentifizierung (wer hat Memory erstellt?)
- ❌ Kein Schutz vor Replay-Angriffen
- ❌ Nur Integrität, keine Authentizität

**Empfehlung:** ⚠️ **Nur für einfache Integritätsprüfung, nicht für Sicherheit**

## Vergleich: Optionen

| Aspekt | Option A (SQLite) | Option B (HMAC) | Option C (Hash) |
|--------|------------------|-----------------|-----------------|
| **Integrität** | ✅ Implizit | ✅ Explizit | ✅ Explizit |
| **Authentifizierung** | ❌ Nein | ✅ Ja | ❌ Nein |
| **Komplexität** | ✅ Niedrig | ⚠️ Mittel | ✅ Niedrig |
| **Secret-Management** | ✅ Nicht nötig | ❌ Erforderlich | ✅ Nicht nötig |
| **Performance** | ✅ Kein Overhead | ⚠️ Minimal | ✅ Minimal |
| **Verteilte Szenarien** | ❌ Nicht geeignet | ✅ Geeignet | ⚠️ Teilweise |
| **Audit-Trail** | ❌ Nein | ✅ Ja | ⚠️ Teilweise |

## Empfehlung

### Für lokale Self-hosted Installationen (aktueller Use-Case)

**Option A (SQLite-Integrität) ist ausreichend:**
- ✅ Lokale Datenbank bietet implizite Integrität
- ✅ Keine zusätzliche Komplexität
- ✅ Keine Secret-Verwaltung nötig
- ✅ Performance-optimal

**Aktueller Status:** ✅ **Implementiert und ausreichend**

### Für verteilte Szenarien (zukünftiger Use-Case)

**Option B (HMAC-Signaturen) wäre sinnvoll:**
- ✅ Explizite Verifizierung möglich
- ✅ Schutz vor Manipulation bei Übertragung
- ✅ Audit-Trail für Signatur-Änderungen
- ✅ Konsistent mit Webhook-Implementierung

**Implementierung:** ⚠️ **Nicht kritisch, kann später hinzugefügt werden**

### Für einfache Integritätsprüfung

**Option C (Content-Hash) wäre möglich:**
- ✅ Einfache Implementierung
- ✅ Keine Secret-Verwaltung
- ⚠️ Aber: Keine Authentifizierung

**Empfehlung:** ❌ **Nicht empfohlen, da Option B besser ist**

## Neutron-Vergleich

**Neutron-Artikel erwähnt:**
> "cryptographically verifiable knowledge units"

**Interpretation:**
- Neutron könnte HMAC-Signaturen für Seeds verwenden
- Oder Content-Hashes für Integrität
- Oder nur Marketing-Sprache für SQLite-Integrität

**Cortex-Status:**
- ✅ Webhooks: HMAC-SHA256 implementiert
- ⚠️ Seeds: SQLite-Integrität (implizit)
- ✅ Konsistent mit lokaler Self-hosted Philosophie

## Fazit

**Aktueller Stand:**
- ✅ **Webhooks:** HMAC-SHA256 Signaturen implementiert
- ✅ **Seeds:** SQLite-Integrität ausreichend für lokale Installationen
- ⚠️ **Explizite Seed-Signierung:** Nicht kritisch, kann später hinzugefügt werden

**Empfehlung:**
- Für **lokale Self-hosted Installationen** (aktueller Use-Case): ✅ **Option A ausreichend**
- Für **verteilte Szenarien** (zukünftiger Use-Case): ⚠️ **Option B könnte sinnvoll sein**
- Für **Audit-Anforderungen**: ⚠️ **Option B empfohlen**

**Priorität:** **Niedrig** - Aktuell nicht kritisch, da Cortex primär für lokale Installationen gedacht ist.

---

**Nächste Schritte (optional):**
- Option B implementieren, wenn verteilte Szenarien benötigt werden
- Secret-Management für HMAC-Signaturen hinzufügen
- Migration bestehender Memories mit Signaturen versehen
