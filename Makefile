.PHONY: help build run test clean install install-binaries kill copy-skill benchmark benchmark-api benchmark-embeddings _benchmark-embeddings-args service-install service-enable service-disable service-start service-stop service-restart service-status service-logs service-reload

help: ## Zeigt diese Hilfe an
	@echo "Cortex Makefile"
	@echo ""
	@echo "Verfügbare Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: build-dashboard ## Baut Dashboard und beide Binaries (cortex-server, cortex-cli)
	go build -o cortex-server ./cmd/cortex-server && \
	go build -o cortex-cli ./cmd/cortex-cli

build-dashboard: ## Baut das React-Dashboard nach internal/dashboard/dist (für Embed). Danach make build für Server mit Dashboard.
	@cd dashboard && npm ci && npm run build

run: ## Startet den Server (mit eingebettetem Dashboard unter /dashboard/)
	go run ./cmd/cortex-server

dev: ## Dev: Vite (Dashboard) und Cortex-Server parallel. Dashboard unter /dashboard/ proxied zu Vite (HMR). CORTEX_ENV=dev
	@echo "Starte Vite (Dashboard) und Cortex-Server mit CORTEX_ENV=dev..."
	@cd dashboard && npm run dev & \
	CORTEX_ENV=dev go run ./cmd/cortex-server; \
	fg 2>/dev/null || true

test: ## Führt alle Tests aus
	go test -v ./...

install-binaries: build ## Installiert Binaries nach /usr/local/bin
	@echo "Installiere Binaries nach /usr/local/bin..." && \
	if [ -w /usr/local/bin ]; then \
		cp cortex-server cortex-cli /usr/local/bin/ && echo "✓ Binaries installiert"; \
	else \
		sudo cp cortex-server cortex-cli /usr/local/bin/ && echo "✓ Binaries installiert (mit sudo)"; \
	fi

install: install-binaries service-install copy-skill ## Vollständige Installation: Binaries, Service und Skills
	@echo "" && \
	echo "=== Installation abgeschlossen ===" && \
	echo "" && \
	echo "Nächste Schritte:" && \
	echo "  make service-enable   # Service beim Login aktivieren" && \
	echo "  make service-start    # Service jetzt starten"

clean: ## Entfernt Build-Artefakte
	rm -f cortex-server cortex-cli coverage.out coverage.html

kill: ## Beendet den Prozess auf dem Cortex-Port (Standard: 9123)
	@PORT=$$([ -f .env ] && grep -E '^CORTEX_PORT=' .env 2>/dev/null | cut -d'=' -f2 | tr -d ' ' || echo "$${CORTEX_PORT:-9123}"); \
	echo "Suche Prozess auf Port $$PORT..."; \
	PID=$$(lsof -ti:$$PORT 2>/dev/null || ss -ltnp 2>/dev/null | grep ":$$PORT " | awk '{print $$6}' | cut -d',' -f2 | cut -d'=' -f2 | head -1); \
	if [ -z "$$PID" ]; then \
		echo "Kein Prozess auf Port $$PORT gefunden"; \
		exit 0; \
	fi; \
	echo "Beende Prozess $$PID..."; \
	kill -9 $$PID 2>/dev/null || kill $$PID 2>/dev/null; \
	echo "✓ Prozess $$PID beendet"

copy-skill: ## Kopiert das Cortex-Skill nach ~/.openclaw/workspace/skills
	@mkdir -p ~/.openclaw/workspace/skills && \
	cp -R skills/cortex/ ~/.openclaw/workspace/skills && \
	echo "✓ Skill kopiert nach ~/.openclaw/workspace/skills/cortex"

benchmark: build ## Führt alle Benchmarks aus (API + Embeddings)
	@echo "=== API Benchmark ===" && \
	$(MAKE) benchmark-api && \
	echo "" && \
	echo "=== Embeddings Benchmark ===" && \
	$(MAKE) benchmark-embeddings

benchmark-api: build ## Führt API-Benchmark aus (benötigt laufenden Server). Usage: make benchmark-api COUNT=50
	@./cortex-cli health > /dev/null 2>&1 || { \
		echo "Fehler: Server läuft nicht auf http://localhost:9123"; \
		echo "Starte Server mit: make run (in separatem Terminal) oder make service-start"; \
		exit 1; \
	}
	@./cortex-cli benchmark $(or $(COUNT),50)

# Helper target to capture positional arguments
_benchmark-embeddings-args:
	@:

benchmark-embeddings: build _benchmark-embeddings-args ## Führt Embedding-Benchmark aus. Usage: make benchmark-embeddings [COUNT] [SERVICE] oder make benchmark-embeddings COUNT=100 SERVICE=both
	@ARGS="$(filter-out benchmark-embeddings _benchmark-embeddings-args,$(MAKECMDGOALS))"; \
	COUNT=$$([ -n "$$ARGS" ] && echo $$ARGS | awk '{print $$1}' || echo "$(or $(COUNT),100)"); \
	SERVICE=$$([ -n "$$ARGS" ] && echo $$ARGS | awk '{print $$2}' || echo "$(or $(SERVICE),both)"); \
	./scripts/benchmark-embeddings.sh $${COUNT} $${SERVICE}

service-install: build ## Installiert systemd User Service-Datei und lädt Konfiguration neu
	@mkdir -p ~/.config/systemd/user && \
	sed "s|%h|$$HOME|g" skills/cortex/cortex-server.service > ~/.config/systemd/user/cortex-server.service && \
	$(MAKE) service-reload && \
	echo "✓ Service-Datei installiert"

service-reload: ## Lädt systemd User-Konfiguration neu
	@systemctl --user daemon-reload && \
	echo "✓ systemd User-Konfiguration neu geladen"

service-enable: service-reload ## Aktiviert den Service beim Login
	@systemctl --user enable cortex-server.service && \
	echo "✓ Service aktiviert"

service-disable: service-reload ## Deaktiviert den Service
	@systemctl --user disable cortex-server.service && \
	echo "✓ Service deaktiviert"

service-start: service-reload ## Startet den Service
	@systemctl --user start cortex-server.service && \
	echo "✓ Service gestartet"

service-stop: service-reload ## Stoppt den Service
	@systemctl --user stop cortex-server.service && \
	echo "✓ Service gestoppt"

service-restart: service-reload ## Startet den Service neu
	@systemctl --user restart cortex-server.service && \
	echo "✓ Service neu gestartet"

service-status: service-reload ## Zeigt den Status des Services
	systemctl --user status cortex-server.service

service-logs: ## Zeigt die Logs des Services (follow mode)
	journalctl --user -u cortex-server.service -f

# Catch-all für Positionsargumente bei benchmark-embeddings
%:
	@if [ "$@" != "benchmark-embeddings" ] && [ "$@" != "_benchmark-embeddings-args" ]; then \
		:; \
	fi

.DEFAULT_GOAL := help
