package models

/*
ContainerInfo representa un contenedor detectado
mediante Docker y su relación con un proceso del Kernel.
*/
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	PID     int
	Project string
	Profile string

	/*
		Process contiene las métricas obtenidas desde
		/proc cuando encontramos el PID del contenedor.
	*/
	Process *ProcessInfo
}

/*
HasKernelProcess indica si el PID principal del
contenedor fue encontrado dentro de la información
expuesta por nuestro módulo Kernel.
*/
func (container ContainerInfo) HasKernelProcess() bool {
	return container.Process != nil
}

/*
IsProjectContainer identifica los contenedores
generados específicamente para el Proyecto 2.
*/
func (container ContainerInfo) IsProjectContainer() bool {
	return container.Project == "202112092"
}
