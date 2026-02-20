#!/bin/bash
# Cortex API Key – anlegen, anzeigen, löschen (CORTEX_API_KEY in .env / Umgebung)
# Usage: ./scripts/api-key.sh create [env_file]
#        ./scripts/api-key.sh delete [env_file]
#        ./scripts/api-key.sh show [env_file]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/.."
source "${SCRIPT_DIR}/lib/common.sh"

VAR_NAME="CORTEX_API_KEY"
KEY_PREFIX="ck_"

# env_file: optional Pfad zu .env; default: Projekt-.env oder aktuelles Verzeichnis
env_file() {
    if [ -n "${1:-}" ]; then
        echo "$1"
    else
        if [ -f "${PROJECT_ROOT}/.env" ]; then
            echo "${PROJECT_ROOT}/.env"
        else
            echo ".env"
        fi
    fi
}

# Generiert einen neuen API-Key (ck_ + 32 Bytes Hex)
generate_key() {
    if command -v openssl &>/dev/null; then
        echo -n "${KEY_PREFIX}$(openssl rand -hex 32)"
    else
        # Fallback: /dev/urandom + od (portabler als xxd)
        echo -n "${KEY_PREFIX}$(head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n')"
    fi
}

cmd_create() {
    local file
    file="$(env_file "${1:-}")"
    local key
    key="$(generate_key)"
    if [ -f "$file" ]; then
        # Entferne alte Zeile CORTEX_API_KEY=...
        if grep -q "^${VAR_NAME}=" "$file" 2>/dev/null; then
            sed "/^${VAR_NAME}=/d" "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
        fi
        echo "${VAR_NAME}=${key}" >> "$file"
        log_success "API-Key in ${file} gesetzt (eine Zeile angehängt)."
    else
        echo "${VAR_NAME}=${key}" > "$file"
        log_success "Datei ${file} erstellt und API-Key gesetzt."
    fi
    echo ""
    log_info "Neuer API-Key (einmalig sichtbar – sicher aufbewahren):"
    echo "  ${key}"
    echo ""
    log_info "Server starten mit: export ${VAR_NAME}=<key>; ./cortex-server"
    log_info "Client/Script: dieselbe Variable setzen oder in .env eintragen."
}

cmd_delete() {
    local file
    file="$(env_file "${1:-}")"
    if [ ! -f "$file" ]; then
        log_warning "Datei ${file} nicht gefunden – nichts zu löschen."
        exit 0
    fi
    if grep -q "^${VAR_NAME}=" "$file" 2>/dev/null; then
        sed "/^${VAR_NAME}=/d" "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
        log_success "API-Key aus ${file} entfernt. Server akzeptiert danach Anfragen ohne Key (wenn Key nicht mehr gesetzt)."
    else
        log_warning "Kein Eintrag ${VAR_NAME} in ${file} gefunden."
    fi
}

cmd_show() {
    local file
    file="$(env_file "${1:-}")"
    if [ ! -f "$file" ]; then
        log_warning "Datei ${file} nicht gefunden."
        exit 0
    fi
    local line
    line="$(grep "^${VAR_NAME}=" "$file" 2>/dev/null || true)"
    if [ -z "$line" ]; then
        log_info "Kein ${VAR_NAME} in ${file} gesetzt."
        exit 0
    fi
    local key
    key="${line#*=}"
    key="${key%%#*}"
    key="$(echo "$key" | tr -d '"' | tr -d "'")"
    if [ -z "$key" ]; then
        log_info "${VAR_NAME} ist in ${file} leer oder auskommentiert."
        exit 0
    fi
    # Nur letzte 4 Zeichen anzeigen (Sicherheit)
    local len="${#key}"
    if [ "$len" -gt 8 ]; then
        log_info "Aktueller Key in ${file} (letzte 4 Zeichen): ...${key: -4}"
    else
        log_info "Key in ${file} gesetzt (Länge ${len})."
    fi
}

cmd_help() {
    cat <<EOF
Cortex API-Key verwalten (${VAR_NAME})

Usage: $0 <command> [env_file]

Commands:
  create [env_file]   Neuen API-Key erzeugen und in env_file setzen (default: .env im Projekt oder aktuelles Verzeichnis)
  delete [env_file]   API-Key aus env_file entfernen
  show [env_file]     Anzeigen, ob ein Key in env_file steht (nur letzte 4 Zeichen)

Beispiele:
  $0 create
  $0 create /path/to/.env
  $0 delete
  $0 show

Der Key wird vom Server nur aus der Umgebungsvariable gelesen. Nach create musst du
den Server neu starten bzw. die Variable exportieren (z. B. in .env des Servers).
EOF
}

case "${1:-}" in
    create)  shift; cmd_create "$@" ;;
    delete)  shift; cmd_delete "$@" ;;
    show)    shift; cmd_show "$@" ;;
    help|--help|-h) cmd_help ;;
    *)
        [ -n "${1:-}" ] && log_error "Unbekannter Befehl: $1"
        cmd_help
        exit 1
        ;;
esac
