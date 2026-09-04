#!/bin/bash

CARNET="202112092"
TOTAL_CONTAINERS=5

IMAGE_HIGH_RAM="so1-202112092-high-ram"
IMAGE_HIGH_CPU="so1-202112092-high-cpu"
IMAGE_LOW="so1-202112092-low"
IMAGE_INTRUDER="so1-202112092-intruder"

echo "=================================================="
echo "Generador de contenedores - Proyecto 2 SO1"
echo "Carnet: ${CARNET}"
echo "Fecha: $(date)"
echo "=================================================="

if ! docker info > /dev/null 2>&1; then
    echo "[ERROR] Docker no se encuentra disponible."
    exit 1
fi

create_high_ram() {
    local name="$1"

    docker run -d \
        --name "$name" \
        --label "so1.project=${CARNET}" \
        --label "so1.profile=high_ram" \
        --label "so1.load=high" \
        --memory="256m" \
        -e MEM_MB=150 \
        "$IMAGE_HIGH_RAM"
}

create_high_cpu() {
    local name="$1"

    docker run -d \
        --name "$name" \
        --label "so1.project=${CARNET}" \
        --label "so1.profile=high_cpu" \
        --label "so1.load=high" \
        --cpus="0.50" \
        "$IMAGE_HIGH_CPU"
}

create_low() {
    local name="$1"

    docker run -d \
        --name "$name" \
        --label "so1.project=${CARNET}" \
        --label "so1.profile=low" \
        --label "so1.load=low" \
        "$IMAGE_LOW"
}

create_intruder() {
    local name="$1"

    docker run -d \
        --name "$name" \
        --label "so1.project=${CARNET}" \
        --label "so1.profile=intruder" \
        --label "so1.load=intruder" \
        "$IMAGE_INTRUDER"
}

for i in $(seq 1 "$TOTAL_CONTAINERS"); do

    PROFILE=$((RANDOM % 4))
    TIMESTAMP=$(date +%s)
    RANDOM_ID=$RANDOM

    case "$PROFILE" in

        0)
            NAME="so1-${CARNET}-high-ram-${TIMESTAMP}-${i}-${RANDOM_ID}"

            echo "[INFO] Creando HIGH_RAM: $NAME"

            create_high_ram "$NAME"
            ;;

        1)
            NAME="so1-${CARNET}-high-cpu-${TIMESTAMP}-${i}-${RANDOM_ID}"

            echo "[INFO] Creando HIGH_CPU: $NAME"

            create_high_cpu "$NAME"
            ;;

        2)
            NAME="so1-${CARNET}-low-${TIMESTAMP}-${i}-${RANDOM_ID}"

            echo "[INFO] Creando LOW: $NAME"

            create_low "$NAME"
            ;;

        3)
            NAME="so1-${CARNET}-intruder-${TIMESTAMP}-${i}-${RANDOM_ID}"

            echo "[INFO] Creando INTRUDER: $NAME"

            create_intruder "$NAME"
            ;;

    esac

    sleep 1
done

echo "=================================================="
echo "Se generaron ${TOTAL_CONTAINERS} contenedores."
echo "Finalización: $(date)"
echo "=================================================="