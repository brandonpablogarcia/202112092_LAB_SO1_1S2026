package services

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"so1.local/daemon202112092/models"
)

const projectCarnet = "202112092"

const (
	imageLow     = "so1-202112092-low"
	imageHighRAM = "so1-202112092-high-ram"
	imageHighCPU = "so1-202112092-high-cpu"
)

/*
shortContainerID reduce el ID para mostrarlo
de manera legible.
*/
func shortContainerID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}

	return id
}

/*
runDockerCommand ejecuta un comando Docker y
devuelve la salida generada.
*/
func runDockerCommand(args ...string) (string, error) {
	cmd := exec.Command(
		"docker",
		args...,
	)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return string(output), fmt.Errorf(
			"docker %s: %w",
			strings.Join(args, " "),
			err,
		)
	}

	return strings.TrimSpace(string(output)), nil
}

/*
createContainer crea automáticamente un contenedor
para corregir faltantes de LOW o HIGH.
*/
func createContainer(
	profile string,
	index int,
) models.ManagementAction {

	timestamp := time.Now().UnixNano()

	name := fmt.Sprintf(
		"so1-%s-%s-daemon-%d-%d",
		projectCarnet,
		strings.ReplaceAll(profile, "_", "-"),
		timestamp,
		index,
	)

	var args []string

	switch profile {

	case "low":

		args = []string{
			"run",
			"-d",
			"--name", name,

			"--label",
			"so1.project=" + projectCarnet,

			"--label",
			"so1.profile=low",

			"--label",
			"so1.source=daemon",

			imageLow,
		}

	case "high_ram":

		args = []string{
			"run",
			"-d",
			"--name", name,

			"--label",
			"so1.project=" + projectCarnet,

			"--label",
			"so1.profile=high_ram",

			"--label",
			"so1.source=daemon",

			"--memory=256m",

			"-e",
			"MEM_MB=150",

			imageHighRAM,
		}

	case "high_cpu":

		args = []string{
			"run",
			"-d",
			"--name", name,

			"--label",
			"so1.project=" + projectCarnet,

			"--label",
			"so1.profile=high_cpu",

			"--label",
			"so1.source=daemon",

			"--cpus=0.50",

			imageHighCPU,
		}

	default:

		return models.ManagementAction{
			Action:        "CREATE",
			ContainerName: name,
			Profile:       profile,
			Reason:        "Perfil no reconocido",
			Success:       false,
			Error:         "perfil de contenedor no soportado",
		}
	}

	output, err := runDockerCommand(
		args...,
	)

	if err != nil {
		return models.ManagementAction{
			Action:        "CREATE",
			ContainerName: name,
			Profile:       profile,
			Reason:        "Correccion de minimo requerido",
			Success:       false,
			Error:         err.Error(),
		}
	}

	return models.ManagementAction{
		Action:        "CREATE",
		ContainerID:   output,
		ContainerName: name,
		Profile:       profile,
		Reason:        "Correccion de minimo requerido",
		Success:       true,
	}
}

/*
ensureMinimumContainers comprueba los mínimos
establecidos por el proyecto.

LOW  >= 3
HIGH >= 2
*/
func ensureMinimumContainers(
	analysis models.ContainerAnalysis,
) (
	[]models.ManagementAction,
	[]models.ManagementAction,
) {

	created := []models.ManagementAction{}
	failed := []models.ManagementAction{}

	/*
		Si faltan LOW, creamos exactamente
		la cantidad necesaria.
	*/
	for i := 0; i < analysis.LowDeficit; i++ {

		action := createContainer(
			"low",
			i+1,
		)

		if action.Success {
			created = append(
				created,
				action,
			)
		} else {
			failed = append(
				failed,
				action,
			)
		}
	}

	/*
		Si faltan HIGH alternamos entre
		high_ram y high_cpu.
	*/
	for i := 0; i < analysis.HighDeficit; i++ {

		profile := "high_ram"

		if i%2 == 1 {
			profile = "high_cpu"
		}

		action := createContainer(
			profile,
			i+1,
		)

		if action.Success {
			created = append(
				created,
				action,
			)
		} else {
			failed = append(
				failed,
				action,
			)
		}
	}

	return created, failed
}

/*
removeContainerCandidate detiene y elimina
un contenedor previamente seleccionado
por el algoritmo de análisis.
*/
func removeContainerCandidate(
	candidate models.ContainerCandidate,
) models.ManagementAction {

	container := candidate.Container

	action := models.ManagementAction{
		Action:        "REMOVE",
		ContainerID:   container.ID,
		ContainerName: container.Name,
		Profile:       container.Profile,
		PID:           container.PID,
		Reason:        candidate.Reason,
	}

	/*
		Segunda barrera de seguridad:
		nunca administramos infraestructura protegida.
	*/
	if isProtectedContainer(container) {

		action.Success = false
		action.Error =
			"contenedor protegido; eliminacion rechazada"

		return action
	}

	/*
		Solamente eliminamos contenedores
		pertenecientes a nuestro proyecto.
	*/
	if !container.IsProjectContainer() {

		action.Success = false
		action.Error =
			"el contenedor no pertenece al proyecto"

		return action
	}

	/*
		Guardamos el PID conocido por el Kernel.

		Este valor será especialmente útil posteriormente
		para relacionar la acción con el evento eBPF.
	*/
	if container.Process != nil {
		action.PID = container.Process.PID
	}

	/*
		Primero solicitamos una terminación controlada.

		Docker intentará detener el proceso principal.
		Posteriormente eBPF auditará estas señales.
	*/
	_, err := runDockerCommand(
		"stop",
		"-t",
		"5",
		container.ID,
	)

	if err != nil {

		action.Success = false
		action.Error = fmt.Sprintf(
			"no se pudo detener el contenedor: %v",
			err,
		)

		return action
	}

	/*
		Una vez detenido, eliminamos el contenedor.
	*/
	_, err = runDockerCommand(
		"rm",
		container.ID,
	)

	if err != nil {

		action.Success = false
		action.Error = fmt.Sprintf(
			"el contenedor fue detenido pero no eliminado: %v",
			err,
		)

		return action
	}

	action.Success = true

	return action
}

/*
ApplyContainerManagement ejecuta la política
autónoma completa.

ORDEN:

1. Corregir faltantes.
2. Procesar candidatos de eliminación.

Corregir los mínimos primero proporciona una
protección adicional antes de eliminar recursos.
*/
func ApplyContainerManagement(
	analysis models.ContainerAnalysis,
) models.ManagementResult {

	result := models.ManagementResult{}

	created, createFailed :=
		ensureMinimumContainers(
			analysis,
		)

	result.Created = append(
		result.Created,
		created...,
	)

	result.Failed = append(
		result.Failed,
		createFailed...,
	)

	/*
		Los candidatos ya fueron calculados respetando
		los mínimos de LOW y HIGH.
	*/
	for _, candidate := range analysis.Candidates {

		action :=
			removeContainerCandidate(
				candidate,
			)

		if action.Success {

			result.Removed = append(
				result.Removed,
				action,
			)

		} else {

			result.Failed = append(
				result.Failed,
				action,
			)
		}
	}

	return result
}
