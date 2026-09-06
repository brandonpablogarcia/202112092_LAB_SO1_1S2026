package services

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"so1.local/daemon202112092/models"
)

/*
dockerInspect representa únicamente las propiedades
que necesitamos de la respuesta de "docker inspect".
*/
type dockerInspect struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`

	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`

	State struct {
		PID     int  `json:"Pid"`
		Running bool `json:"Running"`
	} `json:"State"`
}

/*
GetRunningContainerIDs obtiene los IDs de todos
los contenedores que actualmente están ejecutándose.
*/
func GetRunningContainerIDs() ([]string, error) {
	cmd := exec.Command(
		"docker",
		"ps",
		"-q",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo consultar docker ps: %w",
			err,
		)
	}

	raw := strings.TrimSpace(string(output))

	if raw == "" {
		return []string{}, nil
	}

	ids := strings.Fields(raw)

	return ids, nil
}

/*
InspectRunningContainers obtiene PID, nombre,
imagen y labels de cada contenedor en ejecución.
*/
func InspectRunningContainers() ([]models.ContainerInfo, error) {
	ids, err := GetRunningContainerIDs()
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []models.ContainerInfo{}, nil
	}

	args := append(
		[]string{"inspect"},
		ids...,
	)

	cmd := exec.Command(
		"docker",
		args...,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo ejecutar docker inspect: %w",
			err,
		)
	}

	var inspected []dockerInspect

	if err := json.Unmarshal(output, &inspected); err != nil {
		return nil, fmt.Errorf(
			"no se pudo deserializar docker inspect: %w",
			err,
		)
	}

	containers := make(
		[]models.ContainerInfo,
		0,
		len(inspected),
	)

	for _, item := range inspected {

		if !item.State.Running {
			continue
		}

		name := strings.TrimPrefix(
			item.Name,
			"/",
		)

		project := ""
		profile := ""

		if item.Config.Labels != nil {
			project = item.Config.Labels["so1.project"]
			profile = item.Config.Labels["so1.profile"]
		}

		container := models.ContainerInfo{
			ID:      item.ID,
			Name:    name,
			Image:   item.Config.Image,
			PID:     item.State.PID,
			Project: project,
			Profile: profile,
			Process: nil,
		}

		containers = append(
			containers,
			container,
		)
	}

	return containers, nil
}

/*
MatchContainersWithProcesses relaciona el PID
principal reportado por Docker con el PID encontrado
por el módulo Kernel dentro de /proc.
*/
func MatchContainersWithProcesses(
	containers []models.ContainerInfo,
	processes []models.ProcessInfo,
) []models.ContainerInfo {

	/*
		Creamos un mapa PID -> ProcessInfo para evitar
		recorrer toda la lista de procesos por cada
		contenedor.
	*/
	processByPID := make(
		map[int]models.ProcessInfo,
		len(processes),
	)

	for _, process := range processes {
		processByPID[process.PID] = process
	}

	result := make(
		[]models.ContainerInfo,
		len(containers),
	)

	for i, container := range containers {
		result[i] = container

		process, exists := processByPID[container.PID]

		if exists {
			/*
				Creamos una copia para almacenar
				un puntero estable dentro del contenedor.
			*/
			processCopy := process
			result[i].Process = &processCopy
		}
	}

	return result
}
