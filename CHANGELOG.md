# Changelog

## [Unreleased]

### Added
- ✅ Go Unit-Tests für Store- und Helper-Funktionen
- ✅ Docker-Support (Dockerfile + docker-compose.yml)
- ✅ Strukturiertes Logging mit log/slog
- ✅ CI/CD-Pipeline (GitHub Actions)
- ✅ Projektanalyse-Dokumentation (ANALYSE.md)

### Changed
- 🔄 Logging von `log` zu `log/slog` migriert
- 🔄 Verbesserte Fehlerbehandlung mit strukturierten Logs
- 🔄 API-Key-Authentifizierung entfernt – alle Endpunkte ohne Auth (lokal/Self-hosted)

### Security
- 🔒 Tenant-Isolation (appId / externalUserId)
- 🔒 Keine API-Key-Pflicht (typisch für lokale Nutzung)

## [1.0.0] - Initial Release

### Added
- Go-Server mit SQLite-Backend
- Neutron-kompatible Seeds-API
- Cortex-API (Original)
- Multi-Tenant-Support
- CLI-Tool (cortex-cli.sh)
- E2E-Tests (test-e2e.sh)
- Benchmark-Scripts (benchmark.sh)
