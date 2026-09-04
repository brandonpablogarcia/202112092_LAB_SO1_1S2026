#!/bin/bash

MARKER="SO1_P2_202112092"

echo "Eliminando Cronjob del Proyecto 2..."

CURRENT_CRON=$(crontab -l 2>/dev/null || true)

if echo "$CURRENT_CRON" | grep -q "$MARKER"; then

    echo "$CURRENT_CRON" \
        | grep -v "$MARKER" \
        | crontab -

    echo "Cronjob eliminado correctamente."

else
    echo "No existe un Cronjob del Proyecto 2 instalado."
fi