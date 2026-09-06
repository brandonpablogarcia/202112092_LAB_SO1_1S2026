package models

/*
ManagementAction representa una acción ejecutada
por el gestor autónomo de contenedores.
*/
type ManagementAction struct {
	Action        string
	ContainerID   string
	ContainerName string
	Profile       string
	PID           int
	Reason        string
	Success       bool
	Error         string
}

/*
ManagementResult almacena el resultado completo
de las acciones realizadas durante un ciclo.
*/
type ManagementResult struct {
	Created []ManagementAction
	Removed []ManagementAction
	Failed  []ManagementAction
}
