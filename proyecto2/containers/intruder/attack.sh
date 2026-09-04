#!/bin/sh

echo "=========================================="
echo "CONTENEDOR INTRUSO - SO1"
echo "Carnet: 202112092"
echo "=========================================="

while true
do
    echo "$(date) - Intentando lectura de archivos sensibles"

    cat /etc/passwd > /dev/null 2>&1
    cat /etc/shadow > /dev/null 2>&1 || true
    cat /proc/1/environ > /dev/null 2>&1 || true

    echo "$(date) - Lectura realizada"

    sleep 5
done