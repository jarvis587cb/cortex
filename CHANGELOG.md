# Changelog

## [Unreleased]

### Added
- ✅ Go Unit-Tests für Store- und Helper-Funktionen
- ✅ Docker-Support (Dockerfile + docker-compose.yml)
- ✅ Strukturiertes Logging mit log/slog
- ✅ Optionale API-Key-Authentifizierung
- ✅ CI/CD-Pipeline (GitHub Actions)
- ✅ Projektanalyse-Dokumentation (ANALYSE.md)

### Changed
- 🔄 Logging von `log` zu `log/slog` migriert
- 🔄 Verbesserte Fehlerbehandlung mit strukturierten Logs

### Security
- 🔒 API-Key-Authentifizierung für alle Endpunkte (außer /health)
- 🔒 Tenant-Isolation bereits vorhanden

## [1.0.0] - Initial Release

### Added
- Go-Server mit SQLite-Backend
- Neutron-kompatible Seeds-API
- Cortex-API (Original)
- Multi-Tenant-Support
- CLI-Tool (cortex-cli.sh)
- E2E-Tests (test-e2e.sh)
- Benchmark-Scripts (benchmark.sh)
