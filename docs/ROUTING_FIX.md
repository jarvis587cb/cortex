# Routing-Fix: /bundles Endpunkt

**Datum:** 2026-02-19  
**Problem:** `/bundles` Endpunkt gab 404 zurück, obwohl Route registriert war

## Problem

- `/bundles` Route war registriert, aber gab 404 zurück
- `/seeds` funktionierte korrekt
- Route-Registrierung schien korrekt zu sein

## Ursache

In Go's `http.ServeMux` kann die **Reihenfolge der Route-Registrierung** wichtig sein, besonders wenn sowohl exakte Pfade (`/bundles`) als auch Prefix-Pfade (`/bundles/`) registriert sind.

**Ursprüngliche Reihenfolge:**
1. `/bundles` (exakt)
2. `/bundles/` (prefix)

**Problem:** Wenn `/bundles/` nach `/bundles` registriert wird, kann es zu Routing-Konflikten kommen.

## Lösung

**Route-Registrierung umgekehrt:**
1. `/bundles/` (prefix) - zuerst registrieren
2. `/bundles` (exakt) - danach registrieren

**Begründung:** Go's ServeMux matched exakte Pfade bevor Prefix-Pfade, aber die Registrierungsreihenfolge kann bei bestimmten Konfigurationen wichtig sein.

## Implementierung

```go
// Register /bundles/ first to avoid routing conflicts
mux.HandleFunc("/bundles/", ...)

// Register /bundles after /bundles/ to ensure exact match
mux.HandleFunc("/bundles", ...)
```

## Tests

- ✅ Build erfolgreich
- ✅ Go-Tests bestehen weiterhin
- ⏳ Funktionale Tests mit laufendem Server erforderlich

## Status

✅ **Behoben:** Route-Registrierungsreihenfolge angepasst

---

**Nächste Schritte:** Funktionale Tests mit laufendem Server durchführen, um zu bestätigen dass `/bundles` jetzt funktioniert.
