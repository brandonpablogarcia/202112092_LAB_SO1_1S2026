package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

/*
SystemManager contiene las rutas de los componentes
que deben ser administrados automáticamente por
el Daemon.
*/
type SystemManager struct {
	ProjectDir    string
	KernelDir     string
	CronDir       string
	MonitoringDir string
}

/*
NewSystemManager determina automáticamente la ruta
del proyecto tomando como referencia la ubicación
del ejecutable daemon-so1.

Estructura esperada:

proyecto2/
├── daemon/daemon-so1
├── kernel/
├── cron/
└── monitoring/
*/
func NewSystemManager() (*SystemManager, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo determinar la ruta del ejecutable: %w",
			err,
		)
	}

	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo resolver la ruta del ejecutable: %w",
			err,
		)
	}

	daemonDir := filepath.Dir(executable)
	projectDir := filepath.Dir(daemonDir)

	manager := &SystemManager{
		ProjectDir:    projectDir,
		KernelDir:     filepath.Join(projectDir, "kernel"),
		CronDir:       filepath.Join(projectDir, "cron"),
		MonitoringDir: filepath.Join(projectDir, "monitoring"),
	}

	return manager, nil
}

/*
runCommand ejecuta un comando mostrando directamente
su salida en la terminal del Daemon.
*/
func runCommand(
	directory string,
	command string,
	args ...string,
) error {
	cmd := exec.Command(command, args...)

	cmd.Dir = directory

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"fallo el comando %s: %w",
			command,
			err,
		)
	}

	return nil
}

/*
StartMonitoring inicia Valkey y Grafana mediante
Docker Compose.
*/
func (manager *SystemManager) StartMonitoring() error {
	fmt.Println()
	fmt.Println("[STARTUP] Iniciando Valkey y Grafana...")

	err := runCommand(
		manager.MonitoringDir,
		"docker",
		"compose",
		"up",
		"-d",
	)

	if err != nil {
		return err
	}

	fmt.Println("[OK] Valkey y Grafana iniciados.")

	return nil
}

/*
LoadKernelModule ejecuta el script encargado de
cargar continfo.ko y crear el archivo /proc.
*/
func (manager *SystemManager) LoadKernelModule() error {
	fmt.Println()
	fmt.Println("[STARTUP] Cargando modulo Kernel...")

	err := runCommand(
		manager.KernelDir,
		"/bin/bash",
		"./load_module.sh",
	)

	if err != nil {
		return err
	}

	fmt.Println("[OK] Modulo Kernel disponible.")

	return nil
}

/*
InstallCronJob instala el Cronjob que genera
contenedores cada 2 minutos.
*/
func (manager *SystemManager) InstallCronJob() error {
	fmt.Println()
	fmt.Println("[STARTUP] Instalando Cronjob...")

	err := runCommand(
		manager.CronDir,
		"/bin/bash",
		"./install_cron.sh",
	)

	if err != nil {
		return err
	}

	fmt.Println("[OK] Cronjob instalado.")

	return nil
}

/*
RemoveCronJob elimina el Cronjob cuando
el Daemon finaliza.
*/
func (manager *SystemManager) RemoveCronJob() error {
	fmt.Println()
	fmt.Println("[CLEANUP] Eliminando Cronjob...")

	err := runCommand(
		manager.CronDir,
		"/bin/bash",
		"./remove_cron.sh",
	)

	if err != nil {
		return err
	}

	fmt.Println("[OK] Cronjob eliminado.")

	return nil
}

/*
Startup ejecuta todas las acciones necesarias
antes de iniciar el loop principal.
*/
func (manager *SystemManager) Startup() error {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("INICIALIZANDO COMPONENTES DEL SISTEMA")
	fmt.Println("====================================================")

	fmt.Printf(
		"[INFO] Proyecto: %s\n",
		manager.ProjectDir,
	)

	if err := manager.StartMonitoring(); err != nil {
		return fmt.Errorf(
			"no se pudo iniciar monitoring: %w",
			err,
		)
	}

	if err := manager.LoadKernelModule(); err != nil {
		return fmt.Errorf(
			"no se pudo cargar el modulo Kernel: %w",
			err,
		)
	}

	if err := manager.InstallCronJob(); err != nil {
		return fmt.Errorf(
			"no se pudo instalar el Cronjob: %w",
			err,
		)
	}

	fmt.Println()
	fmt.Println("[OK] Inicializacion completada correctamente.")

	return nil
}

/*
Cleanup ejecuta las operaciones requeridas antes
de finalizar el Daemon.

Por ahora el requisito principal es eliminar el Cronjob.
Más adelante agregaremos la limpieza de eBPF y Valkey.
*/
func (manager *SystemManager) Cleanup() error {
	fmt.Println()
	fmt.Println("====================================================")
	fmt.Println("LIMPIEZA DEL SISTEMA")
	fmt.Println("====================================================")

	if err := manager.RemoveCronJob(); err != nil {
		return err
	}

	fmt.Println("[OK] Limpieza completada correctamente.")

	return nil
}
