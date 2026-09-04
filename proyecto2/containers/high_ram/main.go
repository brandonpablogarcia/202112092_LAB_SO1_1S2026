package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	memMB := 150

	if value := os.Getenv("MEM_MB"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			memMB = parsed
		}
	}

	size := memMB * 1024 * 1024

	fmt.Printf("Contenedor HIGH_RAM iniciado - reservando %d MB\n", memMB)

	memory := make([]byte, size)

	/*
		Se escribe una posición por página para forzar
		que la memoria sea realmente asignada físicamente.
	*/
	for i := 0; i < len(memory); i += 4096 {
		memory[i] = 1
	}

	for {
		/*
			Mantiene la memoria referenciada para evitar
			que sea liberada durante la ejecución.
		*/
		for i := 0; i < len(memory); i += 4096 {
			memory[i]++
		}

		time.Sleep(5 * time.Second)
	}
}