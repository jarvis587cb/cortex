.PHONY: help build run test clean install

help: ## Zeigt diese Hilfe an
	@echo "Cortex Makefile"
	@echo ""
	@echo "Verfügbare Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

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

.DEFAULT_GOAL := help
