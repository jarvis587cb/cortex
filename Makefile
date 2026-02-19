.PHONY: help build run test test-race test-coverage clean fmt vet lint docker-build docker-run docker-stop docker-up docker-down install deps

# Variablen
BINARY_NAME=cortex
CMD_PATH=./cmd/cortex
DOCKER_IMAGE=cortex
DOCKER_TAG=latest
GO_VERSION=1.23
DOCKER_COMPOSE := $(shell command -v docker-compose 2>/dev/null || echo docker compose)

# Farben für Output
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Zeigt diese Hilfe an
	@echo "$(GREEN)Cortex Makefile$(NC)"
	@echo ""
	@echo "Verfügbare Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

build: ## Baut die Binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "$(GREEN)✓ Build erfolgreich$(NC)"

run: ## Startet den Server
	@echo "$(GREEN)Starting Cortex server...$(NC)"
	@go run $(CMD_PATH)

test: ## Führt alle Tests aus
	@echo "$(GREEN)Running tests...$(NC)"
	@go test -v ./...

test-race: ## Führt Tests mit Race-Detection aus
	@echo "$(GREEN)Running tests with race detection...$(NC)"
	@go test -race -v ./...

test-coverage: ## Führt Tests mit Coverage-Report aus
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage-Report erstellt: coverage.html$(NC)"

test-benchmark: ## Führt Benchmark-Tests aus
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...

clean: ## Entfernt Build-Artefakte
	@echo "$(GREEN)Cleaning...$(NC)"
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -f *.test
	@echo "$(GREEN)✓ Clean abgeschlossen$(NC)"

fmt: ## Formatiert den Code
	@echo "$(GREEN)Formatting code...$(NC)"
	@gofmt -w .
	@echo "$(GREEN)✓ Code formatiert$(NC)"

fmt-check: ## Prüft Code-Formatierung ohne Änderungen
	@echo "$(GREEN)Checking code format...$(NC)"
	@test -z $$(gofmt -l .) || (echo "Code ist nicht formatiert. Führe 'make fmt' aus." && exit 1)
	@echo "$(GREEN)✓ Code ist korrekt formatiert$(NC)"

vet: ## Führt go vet aus
	@echo "$(GREEN)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ go vet erfolgreich$(NC)"

lint: vet fmt-check ## Führt alle Linter-Checks aus
	@echo "$(GREEN)✓ Alle Linter-Checks bestanden$(NC)"

deps: ## Aktualisiert Dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	@go mod tidy
	@go mod download
	@echo "$(GREEN)✓ Dependencies aktualisiert$(NC)"

deps-verify: ## Verifiziert Dependencies
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	@go mod verify
	@echo "$(GREEN)✓ Dependencies verifiziert$(NC)"

install: build ## Installiert die Binary (kopiert nach /usr/local/bin)
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✓ $(BINARY_NAME) installiert$(NC)"

# Docker Targets
docker-build: ## Baut Docker-Image
	@echo "$(GREEN)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)✓ Docker image erstellt: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

docker-run: ## Startet Docker-Container (alias: docker-up)
	@echo "$(GREEN)Starting Docker container...$(NC)"
	@$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)✓ Container gestartet$(NC)"

docker-up: docker-run ## Startet Docker-Container (docker compose up -d)

docker-stop: ## Stoppt Docker-Container (alias: docker-down)
	@echo "$(GREEN)Stopping Docker container...$(NC)"
	@$(DOCKER_COMPOSE) down
	@echo "$(GREEN)✓ Container gestoppt$(NC)"

docker-down: docker-stop ## Stoppt Docker-Container (docker compose down)

docker-logs: ## Zeigt Docker-Logs
	@$(DOCKER_COMPOSE) logs -f

docker-clean: ## Entfernt Docker-Images und Container
	@echo "$(GREEN)Cleaning Docker artifacts...$(NC)"
	@$(DOCKER_COMPOSE) down -v
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
	@echo "$(GREEN)✓ Docker-Clean abgeschlossen$(NC)"

# Script-Tests
test-scripts: ## Prüft Bash-Scripts auf Syntax-Fehler
	@echo "$(GREEN)Checking script syntax...$(NC)"
	@bash -n scripts/*.sh
	@echo "$(GREEN)✓ Alle Scripts syntaktisch korrekt$(NC)"

test-e2e: ## Führt E2E-Tests aus (benötigt laufenden Server)
	@echo "$(GREEN)Running E2E tests...$(NC)"
	@./scripts/test-e2e.sh

benchmark: ## Führt Performance-Benchmark aus (benötigt laufenden Server)
	@echo "$(GREEN)Running benchmark...$(NC)"
	@./scripts/benchmark.sh 20

# Development Targets
dev: ## Startet Server im Development-Modus (mit Auto-Reload)
	@echo "$(GREEN)Starting development server...$(NC)"
	@which air > /dev/null || (echo "$(YELLOW)air nicht installiert. Installiere mit: go install github.com/cosmtrek/air@latest$(NC)" && go run $(CMD_PATH))
	@air

check: fmt-check vet test ## Führt alle Checks aus (Format, Vet, Tests)
	@echo "$(GREEN)✓ Alle Checks bestanden$(NC)"

ci: deps-verify fmt-check vet test-race test-coverage ## CI-Pipeline (alle Checks)
	@echo "$(GREEN)✓ CI-Pipeline erfolgreich$(NC)"

# Datenbank-Targets
db-info: ## Zeigt Informationen über die Datenbank
	@echo "$(GREEN)Database Info:$(NC)"
	@echo "  Standard-Pfad: ~/.openclaw/cortex.db"
	@echo "  Aktueller Pfad: $${CORTEX_DB_PATH:-~/.openclaw/cortex.db}"
	@if [ -f "$$HOME/.openclaw/cortex.db" ]; then \
		ls -lh "$$HOME/.openclaw/cortex.db"; \
	else \
		echo "  $(YELLOW)Datenbank existiert noch nicht$(NC)"; \
	fi

db-backup: ## Erstellt Backup der Datenbank
	@echo "$(GREEN)Creating database backup...$(NC)"
	@DB_PATH=$${CORTEX_DB_PATH:-$$HOME/.openclaw/cortex.db}; \
	if [ -f "$$DB_PATH" ]; then \
		cp "$$DB_PATH" "$$DB_PATH.backup.$$(date +%Y%m%d_%H%M%S)"; \
		echo "$(GREEN)✓ Backup erstellt$(NC)"; \
	else \
		echo "$(YELLOW)Datenbank nicht gefunden$(NC)"; \
	fi

# Default Target
.DEFAULT_GOAL := help
