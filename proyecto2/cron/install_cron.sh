#!/bin/bash

PROJECT_DIR="/home/pablo/202112092_LAB_SO1_1S2026/proyecto2"
SCRIPT="${PROJECT_DIR}/containers/scripts/create_containers.sh"
LOG="${PROJECT_DIR}/cron/cronjob.log"
MARKER="SO1_P2_202112092"

CRON_ENTRY="*/2 * * * * /bin/bash ${SCRIPT} >> ${LOG} 2>&1 # ${MARKER}"

echo "Instalando Cronjob del Proyecto 2..."

if crontab -l 2>/dev/null | grep -q "$MARKER"; then
    echo "El Cronjob ya se encuentra instalado."
    exit 0
fi

(
    crontab -l 2>/dev/null
    echo "$CRON_ENTRY"
) | crontab -

echo "Cronjob instalado correctamente."
echo
echo "Entrada instalada:"
crontab -l | grep "$MARKER"