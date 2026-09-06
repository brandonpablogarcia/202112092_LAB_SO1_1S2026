package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"so1.local/daemon202112092/models"
	"so1.local/daemon202112092/services"
)

/*
Intervalo de ejecución del Daemon.

El enunciado permite un intervalo entre
20 y 60 segundos. Utilizamos 30 segundos.
*/
const daemonInterval = 30 * time.Second

/*
kbToMB convierte KB a MB únicamente
para presentar información legible.
*/
func kbToMB(kb uint64) float64 {
	return float64(kb) / 1024.0
}

/*
printTopMemoryProcesses muestra los cinco procesos
con mayor consumo de memoria RSS.
*/
func printTopMemoryProcesses(processes []models.ProcessInfo) {
	copied := make([]models.ProcessInfo, len(processes))
	copy(copied, processes)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].RSSKB > copied[j].RSSKB
	})

	limit := 5

	if len(copied) < limit {
		limit = len(copied)
	}

	fmt.Println()
	fmt.Println("TOP 5 PROCESOS POR RAM")

	for i := 0; i < limit; i++ {
		process := copied[i]

		fmt.Printf(
			"%d. PID=%d | %-20s | RSS=%.2f MB | RAM=%.2f%% | CPU=%.2f%%\n",
			i+1,
			process.PID,
			process.Name,
			kbToMB(process.RSSKB),
			process.MemoryPercent,
			process.CPUPercent,
		)
	}
}

/*
printTopCPUProcesses muestra los cinco procesos
con mayor porcentaje de CPU.
*/
func printTopCPUProcesses(processes []models.ProcessInfo) {
	copied := make([]models.ProcessInfo, len(processes))
	copy(copied, processes)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].CPUPercent > copied[j].CPUPercent
	})

	limit := 5

	if len(copied) < limit {
		limit = len(copied)
	}

	fmt.Println()
	fmt.Println("TOP 5 PROCESOS POR CPU")

	for i := 0; i < limit; i++ {
		process := copied[i]

		fmt.Printf(
			"%d. PID=%d | %-20s | CPU=%.2f%% | RAM=%.2f%%\n",
			i+1,
			process.PID,
			process.Name,
			process.CPUPercent,
			process.MemoryPercent,
		)
	}
}

/*
printContainers muestra la información obtenida
desde Docker junto con las métricas obtenidas
desde nuestro módulo Kernel.
*/
func printContainers(containers []models.ContainerInfo) {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("CONTENEDORES DETECTADOS")
	fmt.Println("====================================================")

	if len(containers) == 0 {
		fmt.Println("[INFO] No existen contenedores en ejecución.")
		return
	}

	for _, container := range containers {

		shortID := container.ID

		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		fmt.Println()

		fmt.Printf(
			"Contenedor: %s\n",
			container.Name,
		)

		fmt.Printf(
			"ID:         %s\n",
			shortID,
		)

		fmt.Printf(
			"Imagen:     %s\n",
			container.Image,
		)

		fmt.Printf(
			"PID Docker: %d\n",
			container.PID,
		)

		if container.Project != "" {
			fmt.Printf(
				"Proyecto:   %s\n",
				container.Project,
			)
		}

		if container.Profile != "" {
			fmt.Printf(
				"Perfil:     %s\n",
				container.Profile,
			)
		}

		if !container.HasKernelProcess() {
			fmt.Println(
				"Kernel Match: NO",
			)

			continue
		}

		process := container.Process

		fmt.Println(
			"Kernel Match: SI",
		)

		fmt.Printf(
			"PID Kernel: %d\n",
			process.PID,
		)

		fmt.Printf(
			"Proceso:    %s\n",
			process.Name,
		)

		fmt.Printf(
			"VSZ:        %.2f MB\n",
			kbToMB(process.VSZKB),
		)

		fmt.Printf(
			"RSS:        %.2f MB\n",
			kbToMB(process.RSSKB),
		)

		fmt.Printf(
			"RAM:        %.2f%%\n",
			process.MemoryPercent,
		)

		fmt.Printf(
			"CPU:        %.2f%%\n",
			process.CPUPercent,
		)
	}
}

/*
printMetricRanking muestra los primeros cinco
contenedores de un ranking.
*/
func printMetricRanking(
	title string,
	containers []models.ContainerInfo,
	metric func(models.ContainerInfo) string,
) {
	fmt.Println()
	fmt.Println(title)

	if len(containers) == 0 {
		fmt.Println("  Sin datos.")
		return
	}

	limit := 5

	if len(containers) < limit {
		limit = len(containers)
	}

	for i := 0; i < limit; i++ {
		container := containers[i]

		fmt.Printf(
			"  %d. %-35s | %s\n",
			i+1,
			container.Name,
			metric(container),
		)
	}
}

/*
printContainerAnalysis muestra clasificación,
restricciones, rankings y candidatos.

IMPORTANTE:
Esta función únicamente muestra decisiones.
No ejecuta docker stop ni docker rm.
*/
func printContainerAnalysis(
	analysis models.ContainerAnalysis,
) {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("ANALISIS DE CONTENEDORES DEL PROYECTO")
	fmt.Println("====================================================")

	fmt.Printf(
		"Total proyecto: %d\n",
		len(analysis.ProjectContainers),
	)

	fmt.Printf(
		"LOW:            %d (minimo requerido: %d)\n",
		len(analysis.Low),
		services.MinLowContainers,
	)

	fmt.Printf(
		"HIGH:           %d (minimo requerido: %d)\n",
		len(analysis.High),
		services.MinHighContainers,
	)

	fmt.Printf(
		"INTRUDER:       %d\n",
		len(analysis.Intruders),
	)

	fmt.Printf(
		"UNKNOWN:        %d\n",
		len(analysis.Unknown),
	)

	fmt.Println()

	if analysis.LowDeficit > 0 {
		fmt.Printf(
			"[ALERTA] Faltan %d contenedores LOW para cumplir el minimo.\n",
			analysis.LowDeficit,
		)
	} else {
		fmt.Println(
			"[OK] Minimo de contenedores LOW satisfecho.",
		)
	}

	if analysis.HighDeficit > 0 {
		fmt.Printf(
			"[ALERTA] Faltan %d contenedores HIGH para cumplir el minimo.\n",
			analysis.HighDeficit,
		)
	} else {
		fmt.Println(
			"[OK] Minimo de contenedores HIGH satisfecho.",
		)
	}

	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("RANKINGS DE CONSUMO")
	fmt.Println("====================================================")

	printMetricRanking(
		"TOP 5 POR RAM %",
		analysis.Rankings.ByRAM,
		func(container models.ContainerInfo) string {
			return fmt.Sprintf(
				"RAM=%.2f%%",
				container.Process.MemoryPercent,
			)
		},
	)

	printMetricRanking(
		"TOP 5 POR VSZ",
		analysis.Rankings.ByVSZ,
		func(container models.ContainerInfo) string {
			return fmt.Sprintf(
				"VSZ=%.2f MB",
				kbToMB(container.Process.VSZKB),
			)
		},
	)

	printMetricRanking(
		"TOP 5 POR RSS",
		analysis.Rankings.ByRSS,
		func(container models.ContainerInfo) string {
			return fmt.Sprintf(
				"RSS=%.2f MB",
				kbToMB(container.Process.RSSKB),
			)
		},
	)

	printMetricRanking(
		"TOP 5 POR CPU %",
		analysis.Rankings.ByCPU,
		func(container models.ContainerInfo) string {
			return fmt.Sprintf(
				"CPU=%.2f%%",
				container.Process.CPUPercent,
			)
		},
	)

	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("CANDIDATOS SELECCIONADOS PARA GESTION")
	fmt.Println("====================================================")

	if len(analysis.Candidates) == 0 {
		fmt.Println(
			"[INFO] No existen candidatos en este ciclo.",
		)
		return
	}

	for index, candidate := range analysis.Candidates {

		fmt.Printf(
			"%d. %s\n",
			index+1,
			candidate.Container.Name,
		)

		fmt.Printf(
			"   Perfil:    %s\n",
			candidate.Container.Profile,
		)

		fmt.Printf(
			"   Motivo:    %s\n",
			candidate.Reason,
		)

		fmt.Printf(
			"   Puntaje:   %d\n",
			candidate.ResourceScore,
		)

		fmt.Printf(
			"   Prioridad: %d\n",
			candidate.Priority,
		)

		if candidate.Container.Process != nil {

			fmt.Printf(
				"   PID:       %d\n",
				candidate.Container.Process.PID,
			)

			fmt.Printf(
				"   RAM:       %.2f%%\n",
				candidate.Container.Process.MemoryPercent,
			)

			fmt.Printf(
				"   VSZ:       %.2f MB\n",
				kbToMB(
					candidate.Container.Process.VSZKB,
				),
			)

			fmt.Printf(
				"   RSS:       %.2f MB\n",
				kbToMB(
					candidate.Container.Process.RSSKB,
				),
			)

			fmt.Printf(
				"   CPU:       %.2f%%\n",
				candidate.Container.Process.CPUPercent,
			)
		}

		fmt.Println()
	}

	fmt.Println(
		"[INFO] Los candidatos anteriores seran procesados por el gestor autonomo.",
	)
}

/*
printManagementResult muestra todas las acciones
reales ejecutadas por el gestor autónomo.
*/
func printManagementResult(
	result models.ManagementResult,
) {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("GESTION AUTONOMA DE CONTENEDORES")
	fmt.Println("====================================================")

	if len(result.Created) == 0 {
		fmt.Println(
			"[INFO] No fue necesario crear contenedores.",
		)
	} else {

		fmt.Println()
		fmt.Println("CONTENEDORES CREADOS")

		for _, action := range result.Created {

			fmt.Printf(
				"[CREATE] %-40s | perfil=%s\n",
				action.ContainerName,
				action.Profile,
			)

			if action.ContainerID != "" {

				id := action.ContainerID

				if len(id) > 12 {
					id = id[:12]
				}

				fmt.Printf(
					"         ID=%s\n",
					id,
				)
			}

			fmt.Printf(
				"         Motivo=%s\n",
				action.Reason,
			)
		}
	}

	if len(result.Removed) == 0 {
		fmt.Println()
		fmt.Println(
			"[INFO] No fue necesario eliminar contenedores.",
		)
	} else {

		fmt.Println()
		fmt.Println("CONTENEDORES ELIMINADOS")

		for _, action := range result.Removed {

			fmt.Printf(
				"[REMOVE] %-40s | perfil=%s\n",
				action.ContainerName,
				action.Profile,
			)

			fmt.Printf(
				"         PID=%d\n",
				action.PID,
			)

			fmt.Printf(
				"         Motivo=%s\n",
				action.Reason,
			)
		}
	}

	if len(result.Failed) > 0 {

		fmt.Println()
		fmt.Println("ACCIONES FALLIDAS")

		for _, action := range result.Failed {

			fmt.Printf(
				"[ERROR] %s | accion=%s | %s\n",
				action.ContainerName,
				action.Action,
				action.Error,
			)
		}

	} else {

		fmt.Println()
		fmt.Println(
			"[OK] Todas las acciones fueron ejecutadas correctamente.",
		)
	}
}

/*
printPostManagementState muestra el estado real
de Docker después de ejecutar las decisiones.
*/
func printPostManagementState(
	analysis models.ContainerAnalysis,
) {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("ESTADO DESPUES DE LA GESTION")
	fmt.Println("====================================================")

	fmt.Printf(
		"LOW:      %d\n",
		len(analysis.Low),
	)

	fmt.Printf(
		"HIGH:     %d\n",
		len(analysis.High),
	)

	fmt.Printf(
		"INTRUDER: %d\n",
		len(analysis.Intruders),
	)

	fmt.Printf(
		"UNKNOWN:  %d\n",
		len(analysis.Unknown),
	)

	if len(analysis.Low) >= services.MinLowContainers {

		fmt.Println(
			"[OK] Restriccion LOW satisfecha.",
		)

	} else {

		fmt.Printf(
			"[ALERTA] LOW insuficientes: %d/%d\n",
			len(analysis.Low),
			services.MinLowContainers,
		)
	}

	if len(analysis.High) >= services.MinHighContainers {

		fmt.Println(
			"[OK] Restriccion HIGH satisfecha.",
		)

	} else {

		fmt.Printf(
			"[ALERTA] HIGH insuficientes: %d/%d\n",
			len(analysis.High),
			services.MinHighContainers,
		)
	}
}

func executeCycle() error {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Printf(
		"CICLO DE TELEMETRIA - %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
	)
	fmt.Println("====================================================")

	fmt.Printf(
		"[INFO] Leyendo %s\n",
		services.ProcPath,
	)

	telemetry, err := services.ReadTelemetry()
	if err != nil {
		return err
	}

	fmt.Println("[OK] Telemetria obtenida correctamente.")

	/*
		Consultamos Docker para obtener todos los
		contenedores que se encuentran ejecutándose.
	*/
	containers, err := services.InspectRunningContainers()
	if err != nil {
		return fmt.Errorf(
			"no se pudieron consultar los contenedores: %w",
			err,
		)
	}

	/*
		Relacionamos el PID principal del contenedor
		con los procesos obtenidos desde el Kernel.
	*/
	containers = services.MatchContainersWithProcesses(
		containers,
		telemetry.Processes,
	)

	/*
		Clasificamos, ordenamos y determinamos
		candidatos SIN eliminarlos.
	*/
	analysis := services.AnalyzeContainers(
		containers,
	)

	fmt.Printf(
		"[OK] Contenedores detectados: %d\n",
		len(containers),
	)

	fmt.Printf(
		"RAM Total:     %.2f MB\n",
		kbToMB(telemetry.Memory.TotalKB),
	)

	fmt.Printf(
		"RAM Libre:     %.2f MB\n",
		kbToMB(telemetry.Memory.FreeKB),
	)

	fmt.Printf(
		"RAM Utilizada: %.2f MB\n",
		kbToMB(telemetry.Memory.UsedKB),
	)

	fmt.Printf(
		"Procesos:      %d\n",
		telemetry.ProcessCount,
	)

	printTopMemoryProcesses(
		telemetry.Processes,
	)

	printTopCPUProcesses(
		telemetry.Processes,
	)

	printContainers(
		containers,
	)

	printContainerAnalysis(
		analysis,
	)

	/*
		Ejecutamos ahora las decisiones reales:

		- Crear contenedores faltantes.
		- Detener candidatos.
		- Eliminar candidatos.
	*/
	managementResult :=
		services.ApplyContainerManagement(
			analysis,
		)

	printManagementResult(
		managementResult,
	)

	/*
		Volvemos a consultar Docker después de realizar
		las acciones para comprobar el estado real.
	*/
	currentContainers, err :=
		services.InspectRunningContainers()

	if err != nil {
		return fmt.Errorf(
			"no se pudo verificar Docker despues de la gestion: %w",
			err,
		)
	}

	/*
		Para verificar cantidades no necesitamos todavía
		las métricas del Kernel de los nuevos procesos.
	*/
	postAnalysis :=
		services.AnalyzeContainers(
			currentContainers,
		)

	printPostManagementState(
		postAnalysis,
	)

	fmt.Println()
	fmt.Printf(
		"[INFO] Proximo ciclo en %v\n",
		daemonInterval,
	)

	return nil
}

func main() {
	fmt.Println("====================================================")
	fmt.Println("DAEMON - PROYECTO 2 SISTEMAS OPERATIVOS 1")
	fmt.Println("Carnet: 202112092")
	fmt.Println("====================================================")

	/*
		El Daemon debe ejecutarse como root porque
		necesita cargar un módulo del Kernel.
	*/
	if os.Geteuid() != 0 {
		fmt.Fprintln(
			os.Stderr,
			"[ERROR] El Daemon necesita privilegios de root.",
		)

		fmt.Fprintln(
			os.Stderr,
			"[INFO] Ejecutelo utilizando: sudo ./daemon-so1",
		)

		os.Exit(1)
	}

	fmt.Println("[OK] Privilegios de root confirmados.")

	fmt.Printf(
		"[INFO] Intervalo configurado: %v\n",
		daemonInterval,
	)

	/*
		Inicializamos el administrador encargado
		de Kernel, Cron y Docker Compose.
	*/
	systemManager, err := services.NewSystemManager()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"[ERROR] %v\n",
			err,
		)

		os.Exit(1)
	}

	/*
		Iniciamos automáticamente los componentes.
	*/
	if err := systemManager.Startup(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"[ERROR] Fallo durante la inicializacion: %v\n",
			err,
		)

		/*
			Intentamos eliminar el Cronjob por seguridad
			aunque la inicialización no haya terminado.
		*/
		_ = systemManager.Cleanup()

		os.Exit(1)
	}

	/*
		defer garantiza que al salir normalmente de main
		intentaremos eliminar el Cronjob.
	*/
	defer func() {
		if err := systemManager.Cleanup(); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"[ERROR] Limpieza incompleta: %v\n",
				err,
			)
		}
	}()

	/*
		Canal para recibir SIGINT y SIGTERM.
	*/
	signalChannel := make(chan os.Signal, 1)

	signal.Notify(
		signalChannel,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	/*
		Primera lectura inmediata.
	*/
	if err := executeCycle(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"[ERROR] Primer ciclo fallido: %v\n",
			err,
		)
	}

	ticker := time.NewTicker(daemonInterval)

	defer ticker.Stop()

	fmt.Println()
	fmt.Println("[OK] Daemon iniciado correctamente.")
	fmt.Println("[INFO] Presiona Ctrl+C para finalizar.")

	for {
		select {

		case <-ticker.C:

			if err := executeCycle(); err != nil {
				fmt.Fprintf(
					os.Stderr,
					"[ERROR] Ciclo fallido: %v\n",
					err,
				)
			}

		case receivedSignal := <-signalChannel:

			fmt.Println()
			fmt.Println("====================================================")
			fmt.Println("FINALIZANDO DAEMON")
			fmt.Println("====================================================")

			fmt.Printf(
				"[INFO] Señal recibida: %s\n",
				receivedSignal.String(),
			)

			/*
				Al retornar, se ejecutará automáticamente
				el defer systemManager.Cleanup().
			*/
			fmt.Println(
				"[INFO] Iniciando limpieza controlada...",
			)

			return
		}
	}
}
