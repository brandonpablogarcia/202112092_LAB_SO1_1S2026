package models

/*
ContainerCandidate representa un contenedor que
el algoritmo considera elegible para eliminación.

IMPORTANTE:
En esta etapa únicamente generamos candidatos.
Todavía no se elimina ningún contenedor.
*/
type ContainerCandidate struct {
	Container     ContainerInfo
	Reason        string
	ResourceScore int
	Priority      int
}

/*
ContainerRankings contiene los cuatro ordenamientos
solicitados para el análisis de contenedores.
*/
type ContainerRankings struct {
	ByRAM []ContainerInfo
	ByVSZ []ContainerInfo
	ByRSS []ContainerInfo
	ByCPU []ContainerInfo
}

/*
ContainerAnalysis representa el resultado completo
del análisis realizado por el Daemon.
*/
type ContainerAnalysis struct {
	ProjectContainers []ContainerInfo

	Low       []ContainerInfo
	High      []ContainerInfo
	Intruders []ContainerInfo
	Unknown   []ContainerInfo

	LowDeficit  int
	HighDeficit int

	Rankings ContainerRankings

	Candidates []ContainerCandidate
}
