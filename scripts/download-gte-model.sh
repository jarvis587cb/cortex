#!/bin/bash
# download-gte-model.sh - Download und Konvertierung des GTE-Small Modells für Cortex
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Standard-Modell-Pfad: ~/.openclaw/gte-small.gtemodel
MODEL_DIR="${HOME}/.openclaw"
MODEL_NAME="gte-small"
GTEMODEL_PATH="${MODEL_DIR}/${MODEL_NAME}.gtemodel"
HF_MODEL_DIR="${MODEL_DIR}/models/${MODEL_NAME}"

echo "=== GTE-Small Modell Download & Konvertierung ==="
echo "Ziel: ${GTEMODEL_PATH}"
echo ""

# Prüfe ob Python verfügbar ist
if ! command -v python3 &> /dev/null; then
    echo "Fehler: python3 ist nicht installiert"
    echo "Bitte installiere Python 3 mit pip"
    exit 1
fi

# Prüfe ob pip verfügbar ist
if ! command -v pip3 &> /dev/null && ! python3 -m pip --version &> /dev/null; then
    echo "Fehler: pip ist nicht installiert"
    echo "Bitte installiere pip: python3 -m ensurepip --upgrade"
    exit 1
fi

# Erstelle/verwende virtuelles Environment für Python-Dependencies
VENV_DIR="${SCRIPT_DIR}/.venv"
VENV_PYTHON="${VENV_DIR}/bin/python"
VENV_PIP="${VENV_DIR}/bin/pip"

echo "Prüfe Python-Dependencies..."

# Erstelle venv falls nicht vorhanden
if [ ! -d "${VENV_DIR}" ]; then
    echo "Erstelle virtuelles Python-Environment..."
    python3 -m venv "${VENV_DIR}" || {
        echo "Fehler beim Erstellen des virtuellen Environments"
        exit 1
    }
fi

# Installiere Pakete im venv falls nötig
"${VENV_PYTHON}" -c "import safetensors, requests, numpy" 2>/dev/null || {
    echo "Installiere safetensors, requests und numpy im venv..."
    "${VENV_PIP}" install safetensors requests numpy || {
        echo "Fehler beim Installieren der Python-Dependencies"
        exit 1
    }
}

# Erstelle Modell-Verzeichnis
mkdir -p "${HF_MODEL_DIR}"

# Download von Hugging Face falls noch nicht vorhanden
echo ""
echo "Lade Modell-Dateien von Hugging Face..."
BASE_URL="https://huggingface.co/thenlper/gte-small/resolve/main"
FILES=("config.json" "vocab.txt" "tokenizer_config.json" "special_tokens_map.json" "model.safetensors")

for file in "${FILES[@]}"; do
    file_path="${HF_MODEL_DIR}/${file}"
    if [ -f "${file_path}" ]; then
        echo "  ✓ ${file} bereits vorhanden"
    else
        echo "  ↓ Lade ${file}..."
        url="${BASE_URL}/${file}"
        curl -L -f -o "${file_path}" "${url}" || {
            echo "Fehler beim Download von ${file}"
            exit 1
        }
    fi
done

# Konvertiere zu .gtemodel Format
echo ""
echo "Konvertiere Modell zu .gtemodel Format..."

# Prüfe ob convert_model.py existiert (aus gte-go repo)
CONVERT_SCRIPT="${PROJECT_ROOT}/scripts/convert_model.py"
if [ ! -f "${CONVERT_SCRIPT}" ]; then
    echo "Lade convert_model.py von gte-go..."
    curl -L -f -o "${CONVERT_SCRIPT}" "https://raw.githubusercontent.com/rcarmo/gte-go/main/convert_model.py" || {
        echo "Fehler beim Download von convert_model.py"
        exit 1
    }
fi

# Führe Konvertierung aus (verwende venv-Python)
"${VENV_PYTHON}" "${CONVERT_SCRIPT}" "${HF_MODEL_DIR}" "${GTEMODEL_PATH}" || {
    echo "Fehler bei der Modell-Konvertierung"
    exit 1
}

echo ""
echo "✓ Modell erfolgreich heruntergeladen und konvertiert!"
echo ""
echo "Modell-Pfad: ${GTEMODEL_PATH}"
echo ""
echo "Um das Modell zu verwenden, setze in deiner .env:"
echo "  CORTEX_EMBEDDING_MODEL_PATH=${GTEMODEL_PATH}"
echo ""
echo "Oder verwende den Standard-Pfad:"
echo "  CORTEX_EMBEDDING_MODEL_PATH=~/.openclaw/gte-small.gtemodel"
echo ""
