package services

import (
	"sort"
	"strings"

	"so1.local/daemon202112092/models"
)

const (
	/*
		Restricciones mínimas establecidas
		para los contenedores del proyecto.
	*/
	MinLowContainers  = 3
	MinHighContainers = 2
)

/*
isProtectedContainer protege explícitamente los
componentes de infraestructura.

Aunque Grafana y Valkey no utilizan la etiqueta
so1.project=202112092, agregamos esta protección
como una segunda barrera de seguridad.
*/
func isProtectedContainer(
	container models.ContainerInfo,
) bool {
	name := strings.ToLower(container.Name)

	return strings.Contains(name, "grafana") ||
		strings.Contains(name, "valkey")
}

/*
isHighProfile determina si un contenedor pertenece
al grupo de alto consumo.

Para nuestro proyecto:

high_ram + high_cpu = HIGH
*/
func isHighProfile(profile string) bool {
	return profile == "high_ram" ||
		profile == "high_cpu"
}

/*
containersWithMetrics elimina del ranking aquellos
contenedores que no pudieron relacionarse con un
proceso del Kernel.

No significa que se eliminen del sistema.
Simplemente no tomaremos decisiones sobre ellos
sin disponer de métricas válidas.
*/
func containersWithMetrics(
	containers []models.ContainerInfo,
) []models.ContainerInfo {
	result := make(
		[]models.ContainerInfo,
		0,
		len(containers),
	)

	for _, container := range containers {
		if container.Process != nil {
			result = append(
				result,
				container,
			)
		}
	}

	return result
}

/*
sortByRAM ordena de mayor a menor porcentaje
de memoria RAM.
*/
func sortByRAM(
	containers []models.ContainerInfo,
) []models.ContainerInfo {
	result := append(
		[]models.ContainerInfo(nil),
		containers...,
	)

	sort.SliceStable(
		result,
		func(i, j int) bool {
			return result[i].Process.MemoryPercent >
				result[j].Process.MemoryPercent
		},
	)

	return result
}

/*
sortByVSZ ordena de mayor a menor memoria virtual.
*/
func sortByVSZ(
	containers []models.ContainerInfo,
) []models.ContainerInfo {
	result := append(
		[]models.ContainerInfo(nil),
		containers...,
	)

	sort.SliceStable(
		result,
		func(i, j int) bool {
			return result[i].Process.VSZKB >
				result[j].Process.VSZKB
		},
	)

	return result
}

/*
sortByRSS ordena de mayor a menor memoria física.
*/
func sortByRSS(
	containers []models.ContainerInfo,
) []models.ContainerInfo {
	result := append(
		[]models.ContainerInfo(nil),
		containers...,
	)

	sort.SliceStable(
		result,
		func(i, j int) bool {
			return result[i].Process.RSSKB >
				result[j].Process.RSSKB
		},
	)

	return result
}

/*
sortByCPU ordena de mayor a menor porcentaje
de CPU.
*/
func sortByCPU(
	containers []models.ContainerInfo,
) []models.ContainerInfo {
	result := append(
		[]models.ContainerInfo(nil),
		containers...,
	)

	sort.SliceStable(
		result,
		func(i, j int) bool {
			return result[i].Process.CPUPercent >
				result[j].Process.CPUPercent
		},
	)

	return result
}

/*
buildResourceScores construye un puntaje combinado
utilizando las cuatro clasificaciones.

Un contenedor recibe mayor puntaje mientras más
arriba aparezca en los rankings de:

- RAM
- VSZ
- RSS
- CPU

Esta es nuestra estrategia de decisión porque el
enunciado solicita utilizar las cuatro métricas,
pero no especifica una fórmula matemática concreta.
*/
func buildResourceScores(
	rankings models.ContainerRankings,
) map[string]int {
	scores := make(map[string]int)

	addRanking := func(
		containers []models.ContainerInfo,
	) {
		total := len(containers)

		for index, container := range containers {
			/*
				El primer lugar obtiene más puntos.
			*/
			points := total - index

			scores[container.ID] += points
		}
	}

	addRanking(rankings.ByRAM)
	addRanking(rankings.ByVSZ)
	addRanking(rankings.ByRSS)
	addRanking(rankings.ByCPU)

	return scores
}

/*
selectResourceCandidates selecciona únicamente
la cantidad de contenedores que excede el mínimo.

Ejemplo:

LOW existentes: 5
LOW mínimo:     3

Candidatos LOW: 2
*/
func selectResourceCandidates(
	group []models.ContainerInfo,
	amount int,
	scores map[string]int,
	reason string,
	priority int,
) []models.ContainerCandidate {
	if amount <= 0 {
		return []models.ContainerCandidate{}
	}

	valid := containersWithMetrics(group)

	sort.SliceStable(
		valid,
		func(i, j int) bool {
			return scores[valid[i].ID] >
				scores[valid[j].ID]
		},
	)

	if amount > len(valid) {
		amount = len(valid)
	}

	result := make(
		[]models.ContainerCandidate,
		0,
		amount,
	)

	for i := 0; i < amount; i++ {
		result = append(
			result,
			models.ContainerCandidate{
				Container:     valid[i],
				Reason:        reason,
				ResourceScore: scores[valid[i].ID],
				Priority:      priority,
			},
		)
	}

	return result
}

/*
AnalyzeContainers realiza toda la clasificación,
ordenamiento y generación de candidatos.

NO elimina contenedores.
*/
func AnalyzeContainers(
	containers []models.ContainerInfo,
) models.ContainerAnalysis {
	analysis := models.ContainerAnalysis{}

	/*
		Primero aislamos únicamente los contenedores
		pertenecientes al Proyecto 2.
	*/
	for _, container := range containers {

		if isProtectedContainer(container) {
			continue
		}

		if !container.IsProjectContainer() {
			continue
		}

		analysis.ProjectContainers = append(
			analysis.ProjectContainers,
			container,
		)

		switch {

		case container.Profile == "low":

			analysis.Low = append(
				analysis.Low,
				container,
			)

		case isHighProfile(container.Profile):

			analysis.High = append(
				analysis.High,
				container,
			)

		case container.Profile == "intruder":

			analysis.Intruders = append(
				analysis.Intruders,
				container,
			)

		default:

			analysis.Unknown = append(
				analysis.Unknown,
				container,
			)
		}
	}

	/*
		Calculamos si faltan contenedores para cumplir
		los mínimos establecidos.
	*/
	if len(analysis.Low) < MinLowContainers {
		analysis.LowDeficit =
			MinLowContainers - len(analysis.Low)
	}

	if len(analysis.High) < MinHighContainers {
		analysis.HighDeficit =
			MinHighContainers - len(analysis.High)
	}

	/*
		Los rankings solo incluyen contenedores que
		tienen correspondencia con el Kernel.
	*/
	withMetrics := containersWithMetrics(
		analysis.ProjectContainers,
	)

	analysis.Rankings = models.ContainerRankings{
		ByRAM: sortByRAM(withMetrics),
		ByVSZ: sortByVSZ(withMetrics),
		ByRSS: sortByRSS(withMetrics),
		ByCPU: sortByCPU(withMetrics),
	}

	scores := buildResourceScores(
		analysis.Rankings,
	)

	/*
		Los intrusos se consideran candidatos
		prioritarios porque representan la carga
		maliciosa simulada del proyecto.
	*/
	for _, container := range analysis.Intruders {

		if container.Process == nil {
			continue
		}

		analysis.Candidates = append(
			analysis.Candidates,
			models.ContainerCandidate{
				Container:     container,
				Reason:        "Contenedor intruso detectado",
				ResourceScore: scores[container.ID],
				Priority:      100,
			},
		)
	}

	/*
		LOW:

		Solamente se pueden marcar como candidatos
		los que exceden el mínimo de 3.
	*/
	lowExcess :=
		len(analysis.Low) - MinLowContainers

	analysis.Candidates = append(
		analysis.Candidates,
		selectResourceCandidates(
			analysis.Low,
			lowExcess,
			scores,
			"Exceso de contenedores LOW",
			50,
		)...,
	)

	/*
		HIGH:

		high_ram y high_cpu pertenecen al mismo grupo.
		Solamente podemos marcar los que exceden
		el mínimo combinado de 2.
	*/
	highExcess :=
		len(analysis.High) - MinHighContainers

	analysis.Candidates = append(
		analysis.Candidates,
		selectResourceCandidates(
			analysis.High,
			highExcess,
			scores,
			"Exceso de contenedores HIGH",
			60,
		)...,
	)

	/*
		Ordenamos candidatos:

		1. Prioridad.
		2. Consumo combinado.
	*/
	sort.SliceStable(
		analysis.Candidates,
		func(i, j int) bool {

			if analysis.Candidates[i].Priority !=
				analysis.Candidates[j].Priority {

				return analysis.Candidates[i].Priority >
					analysis.Candidates[j].Priority
			}

			return analysis.Candidates[i].ResourceScore >
				analysis.Candidates[j].ResourceScore
		},
	)

	return analysis
}
