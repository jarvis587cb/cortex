# Authentifizierung – API-Key entfernt

**Datum:** 2026-02-19  
**Status:** API-Key-Authentifizierung wurde aus dem Projekt entfernt.

Cortex verwendet **keine** API-Key-Authentifizierung. Alle Endpunkte sind ohne Auth erreichbar (typisch für lokale Self-hosted-Nutzung).

- Es gibt keine Umgebungsvariable `CORTEX_API_KEY` mehr.
- Das SDK hat keine `apiKey`-Option mehr.
- curl-Beispiele benötigen keinen Auth-Header.

Dieses Dokument bleibt als Referenz für die Historie (zuvor gab es optionalen X-API-Key/Authorization-Support).
