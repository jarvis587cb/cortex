# Authentifizierung-Fix: X-API-Key Support

**Datum:** 2026-02-19  
**Problem:** Inkonsistenz zwischen SDK/Dokumentation (`X-API-Key`) und Middleware (`Authorization`)

## Problem

- **SDK verwendet:** `X-API-Key` Header
- **Dokumentation zeigt:** `X-API-Key` Header
- **Middleware erwartete:** `Authorization` Header

**Ergebnis:** SDK konnte nicht authentifizieren, wenn `CORTEX_API_KEY` gesetzt war.

## Lösung

Middleware wurde angepasst, um **beide Header-Formate** zu unterstützen:

1. **X-API-Key Header** (Priorität) - wie im SDK verwendet
2. **Authorization Header** (Fallback) - für Rückwärtskompatibilität
   - `Authorization: Bearer <key>`
   - `Authorization: <key>` (direkt)

## Implementierung

```go
// Support both X-API-Key header and Authorization header
providedKey := r.Header.Get("X-API-Key")
if providedKey == "" {
    // Fallback to Authorization header
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        // Return 401 Unauthorized
    }
    // Support both "Bearer <key>" and direct key
    providedKey = strings.TrimPrefix(authHeader, "Bearer ")
    providedKey = strings.TrimSpace(providedKey)
}
```

## Tests

Neue Tests hinzugefügt:
- ✅ `X-API-Key` Header funktioniert
- ✅ `Authorization: Bearer` Header funktioniert
- ✅ `Authorization` direkt funktioniert
- ✅ Fehlender Header gibt 401
- ✅ Falscher Key gibt 401
- ✅ Ohne `CORTEX_API_KEY` ist Auth deaktiviert

## Verwendung

### SDK (automatisch)
```typescript
const client = new CortexClient({
    baseUrl: "http://localhost:9123",
    apiKey: "your-key" // Wird als X-API-Key gesendet
});
```

### curl (beide Formate möglich)
```bash
# X-API-Key (empfohlen)
curl -H "X-API-Key: your-key" http://localhost:9123/seeds

# Authorization Bearer (auch möglich)
curl -H "Authorization: Bearer your-key" http://localhost:9123/seeds
```

## Status

✅ **Behoben:** Middleware unterstützt jetzt beide Header-Formate  
✅ **Getestet:** Alle Auth-Tests bestehen  
✅ **Kompatibel:** SDK funktioniert jetzt korrekt mit Auth

---

**Nächste Schritte:** SDK kann jetzt korrekt mit gesetztem `CORTEX_API_KEY` verwendet werden.
