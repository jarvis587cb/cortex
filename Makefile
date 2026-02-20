.PHONY: help build run test clean install kill service-install service-enable service-disable service-start service-stop service-restart service-status service-logs service-reload

help: ## Zeigt diese Hilfe an
	@echo "Cortex Makefile"
	@echo ""
	@echo "Verfügbare Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Baut beide Binaries (cortex-server, cortex-cli)
	go build -o cortex-server ./cmd/cortex-server
	go build -o cortex-cli ./cmd/cortex-cli

run: ## Startet den Server
	go run ./cmd/cortex-server

test: ## Führt alle Tests aus
	go test -v ./...

install: build ## Installiert beide Binaries nach /usr/local/bin
	sudo cp cortex-server cortex-cli /usr/local/bin/

clean: ## Entfernt Build-Artefakte
	rm -f cortex-server cortex-cli coverage.out coverage.html

kill: ## Beendet den Prozess auf dem Cortex-Port (Standard: 9123)
	@if [ -f .env ]; then \
		PORT=$$(grep -E '^CORTEX_PORT=' .env 2>/dev/null | cut -d'=' -f2 | tr -d ' ' || echo "9123"); \
	else \
		PORT=$${CORTEX_PORT:-9123}; \
	fi; \
	echo "Suche Prozess auf Port $$PORT..."; \
	PID=$$(lsof -ti:$$PORT 2>/dev/null); \
	if [ -z "$$PID" ]; then \
		PID=$$(ss -ltnp 2>/dev/null | grep ":$$PORT " | awk '{print $$6}' | cut -d',' -f2 | cut -d'=' -f2 | head -1); \
	fi; \
	if [ -z "$$PID" ]; then \
		PID=$$(fuser $$PORT/tcp 2>/dev/null | awk '{print $$1}' | head -1); \
	fi; \
	if [ -z "$$PID" ]; then \
		echo "Kein Prozess auf Port $$PORT gefunden"; \
		exit 0; \
	fi; \
	echo "Beende Prozess $$PID auf Port $$PORT..."; \
	kill -9 $$PID 2>/dev/null || kill $$PID 2>/dev/null || (echo "Fehler beim Beenden des Prozesses"; exit 1); \
	echo "Prozess $$PID beendet"

service-install: build ## Installiert systemd User Service-Datei
	@mkdir -p ~/.config/systemd/user
	@sed "s|%h|$$HOME|g" skills/cortex/cortex-server.service > ~/.config/systemd/user/cortex-server.service
	@echo "Service-Datei installiert nach ~/.config/systemd/user/cortex-server.service"
	@$(MAKE) service-reload

service-reload: ## Lädt systemd User-Konfiguration neu
	systemctl --user daemon-reload
	@echo "systemd User-Konfiguration neu geladen"

service-enable: service-reload ## Aktiviert den Service beim Login
	systemctl --user enable cortex-server.service
	@echo "Service aktiviert"

service-disable: service-reload ## Deaktiviert den Service
	systemctl --user disable cortex-server.service
	@echo "Service deaktiviert"

service-start: service-reload ## Startet den Service
	systemctl --user start cortex-server.service
	@echo "Service gestartet"

service-stop: service-reload ## Stoppt den Service
	systemctl --user stop cortex-server.service
	@echo "Service gestoppt"

service-restart: service-reload ## Startet den Service neu
	systemctl --user restart cortex-server.service
	@echo "Service neu gestartet"

service-status: service-reload ## Zeigt den Status des Services
	systemctl --user status cortex-server.service

service-logs: ## Zeigt die Logs des Services (follow mode)
	journalctl --user -u cortex-server.service -f

.DEFAULT_GOAL := help
