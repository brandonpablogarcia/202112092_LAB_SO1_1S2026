#!/bin/bash

set -e

MODULE_NAME="continfo"
PROC_FILE="/proc/continfo_pr2_so1_202112092"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_FILE="${SCRIPT_DIR}/continfo.ko"

if [ "$EUID" -eq 0 ]; then
    SUDO=""
else
    SUDO="sudo"
fi

echo "=========================================="
echo "Cargando modulo Kernel - SO1"
echo "Carnet: 202112092"
echo "=========================================="

if lsmod | grep -q "^${MODULE_NAME}"; then
    echo "[INFO] El modulo ya estaba cargado."
    echo "[INFO] Descargando version anterior..."

    $SUDO rmmod "$MODULE_NAME"
fi

if [ ! -f "$MODULE_FILE" ]; then
    echo "[ERROR] No existe $MODULE_FILE"
    echo "[INFO] Ejecute 'make' dentro de kernel/ primero."
    exit 1
fi

$SUDO insmod "$MODULE_FILE"

if [ ! -r "$PROC_FILE" ]; then
    echo "[ERROR] El modulo se cargo, pero no existe $PROC_FILE"
    exit 1
fi

echo "[OK] Modulo cargado correctamente."
echo "[OK] Archivo disponible:"
echo "$PROC_FILE"

echo
echo "Resumen de telemetria:"

if command -v jq > /dev/null 2>&1; then
    cat "$PROC_FILE" \
        | jq '{memory: .memory, process_count: .process_count}'
else
    echo "[INFO] Modulo cargado y archivo /proc disponible."
fi