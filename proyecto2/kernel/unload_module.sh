#!/bin/bash

set -e

MODULE_NAME="continfo"
PROC_FILE="/proc/continfo_pr2_so1_202112092"

if [ "$EUID" -eq 0 ]; then
    SUDO=""
else
    SUDO="sudo"
fi

echo "=========================================="
echo "Descargando modulo Kernel - SO1"
echo "Carnet: 202112092"
echo "=========================================="

if lsmod | grep -q "^${MODULE_NAME}"; then
    $SUDO rmmod "$MODULE_NAME"

    echo "[OK] Modulo descargado correctamente."
else
    echo "[INFO] El modulo no estaba cargado."
fi

if [ -e "$PROC_FILE" ]; then
    echo "[ERROR] $PROC_FILE continua existiendo."
    exit 1
fi

echo "[OK] Archivo /proc eliminado correctamente."