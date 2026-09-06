package services

import (
	"encoding/json"
	"fmt"
	"os"

	"so1.local/daemon202112092/models"
)

const ProcPath = "/proc/continfo_pr2_so1_202112092"

/*
ReadTelemetry lee el archivo /proc generado por
el módulo Kernel y deserializa el JSON hacia
estructuras Go.
*/
func ReadTelemetry() (*models.Telemetry, error) {
	data, err := os.ReadFile(ProcPath)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo leer %s: %w",
			ProcPath,
			err,
		)
	}

	var telemetry models.Telemetry

	err = json.Unmarshal(data, &telemetry)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo deserializar la telemetria: %w",
			err,
		)
	}

	return &telemetry, nil
}
