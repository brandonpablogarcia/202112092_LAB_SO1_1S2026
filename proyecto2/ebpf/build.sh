#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")" &&
    pwd
)"

cd "$SCRIPT_DIR"

VMLINUX="/sys/kernel/btf/vmlinux"

echo "=========================================="
echo "COMPILACION eBPF - PROYECTO 2 SO1"
echo "Carnet: 202112092"
echo "=========================================="

if [ ! -r "$VMLINUX" ]; then
    echo "[ERROR] No existe BTF en:"
    echo "$VMLINUX"
    exit 1
fi

echo "[INFO] Generando vmlinux.h..."

bpftool btf dump \
    file "$VMLINUX" \
    format c \
    > vmlinux.h

ARCH="$(uname -m)"

case "$ARCH" in

    x86_64)
        TARGET_ARCH="x86"
        ARCH_INCLUDE="/usr/include/x86_64-linux-gnu"
        ;;

    aarch64)
        TARGET_ARCH="arm64"
        ARCH_INCLUDE="/usr/include/aarch64-linux-gnu"
        ;;

    *)
        echo "[ERROR] Arquitectura no soportada: $ARCH"
        exit 1
        ;;

esac

echo "[INFO] Arquitectura: $ARCH"
echo "[INFO] Compilando kill_monitor.bpf.c..."

clang \
    -O2 \
    -g \
    -target bpf \
    -D__TARGET_ARCH_${TARGET_ARCH} \
    -I. \
    -I/usr/include \
    -I/usr/include/bpf \
    -I"$ARCH_INCLUDE" \
    -c kill_monitor.bpf.c \
    -o kill_monitor.bpf.o

echo
echo "[OK] Programa eBPF compilado:"
ls -lh kill_monitor.bpf.o