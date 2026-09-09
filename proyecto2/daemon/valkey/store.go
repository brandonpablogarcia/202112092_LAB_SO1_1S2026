package valkeystore

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	valkey "github.com/valkey-io/valkey-go"

	"so1.local/daemon202112092/models"
)

const (
	KeySystemCurrent     = "so1:202112092:system:current"
	KeySystemHistory     = "so1:202112092:system:history"
	KeyTopRAMCurrent     = "so1:202112092:top:ram:current"
	KeyTopCPUCurrent     = "so1:202112092:top:cpu:current"
	KeyContainersCurrent = "so1:202112092:containers:current"
	KeyManagementCurrent = "so1:202112092:management:current"
	KeyDeletedEvents     = "so1:202112092:events:deleted"
	KeyDeletedCount      = "so1:202112092:events:deleted:count"
	KeyDeletedCurrent    = "so1:202112092:events:deleted:current"
)

/*
Store administra la conexión del Daemon
con la base de datos Valkey.
*/
type Store struct {
	client valkey.Client
	ctx    context.Context
}

/*
SystemSnapshot representa una muestra completa
del estado del sistema en un instante determinado.
*/
type SystemSnapshot struct {
	Timestamp         string            `json:"timestamp"`
	TimestampUnix     int64             `json:"timestamp_unix"`
	Memory            models.MemoryInfo `json:"memory"`
	ProcessCount      int               `json:"process_count"`
	ProjectContainers int               `json:"project_containers"`
	LowContainers     int               `json:"low_containers"`
	HighContainers    int               `json:"high_containers"`
	Intruders         int               `json:"intruders"`
}

/*
RankingItem contiene únicamente los datos necesarios
para los rankings que posteriormente utilizará Grafana.
*/
type RankingItem struct {
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	PID           int     `json:"pid"`
	Profile       string  `json:"profile"`
	RAMPercent    float64 `json:"ram_percent"`
	CPUPercent    float64 `json:"cpu_percent"`
	VSZKB         uint64  `json:"vsz_kb"`
	RSSKB         uint64  `json:"rss_kb"`
}

/*
ContainerSnapshot representa un contenedor activo.
*/
type ContainerSnapshot struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Profile string `json:"profile"`
}

/*
ManagementSnapshot registra información de la gestión
del ciclo.

IMPORTANTE:
removed_requested NO significa eliminación confirmada.
La confirmación real se añadirá posteriormente mediante eBPF.
*/
type ManagementSnapshot struct {
	Timestamp        string `json:"timestamp"`
	Created          int    `json:"created"`
	RemovalRequested int    `json:"removal_requested"`
	FailedActions    int    `json:"failed_actions"`
}

/*
DeletedEvent representa una eliminación que fue
confirmada mediante un evento capturado por eBPF.

A diferencia de removal_requested, este registro
SÍ representa una confirmación proveniente del Kernel.
*/
type DeletedEvent struct {
	Timestamp     string `json:"timestamp"`
	TimestampUnix int64  `json:"timestamp_unix"`

	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Profile       string `json:"profile"`

	TargetPID int `json:"target_pid"`

	SenderPID  uint32 `json:"sender_pid"`
	SenderTGID uint32 `json:"sender_tgid"`

	Signal     int32  `json:"signal"`
	SignalName string `json:"signal_name"`

	SenderComm string `json:"sender_comm"`

	Reason string `json:"reason"`
}

/*
NewStore crea una conexión con Valkey.
*/
func NewStore(address string) (*Store, error) {
	client, err := valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{address},
		},
	)

	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo crear el cliente Valkey: %w",
			err,
		)
	}

	return &Store{
		client: client,
		ctx:    context.Background(),
	}, nil
}

/*
Ping comprueba que Valkey realmente responde.
*/
func (store *Store) Ping() error {
	err := store.client.Do(
		store.ctx,
		store.client.B().
			Ping().
			Build(),
	).Error()

	if err != nil {
		return fmt.Errorf(
			"Valkey no responde: %w",
			err,
		)
	}

	return nil
}

/*
WaitUntilReady espera a que Valkey termine
de inicializarse.
*/
func (store *Store) WaitUntilReady(
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)

	for {
		if err := store.Ping(); err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf(
				"Valkey no estuvo disponible despues de %v",
				timeout,
			)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

/*
Close libera la conexión al finalizar el Daemon.
*/
func (store *Store) Close() {
	store.client.Close()
}

/*
setJSON serializa una estructura Go y la almacena
como JSON dentro de Valkey.
*/
func (store *Store) setJSON(
	key string,
	value interface{},
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf(
			"no se pudo serializar %s: %w",
			key,
			err,
		)
	}

	err = store.client.Do(
		store.ctx,
		store.client.B().
			Set().
			Key(key).
			Value(string(data)).
			Build(),
	).Error()

	if err != nil {
		return fmt.Errorf(
			"no se pudo guardar %s en Valkey: %w",
			key,
			err,
		)
	}

	return nil
}

/*
buildRanking convierte el ranking del analizador
en una estructura reducida para almacenar en Valkey.
*/
func buildRanking(
	containers []models.ContainerInfo,
) []RankingItem {
	limit := 5

	if len(containers) < limit {
		limit = len(containers)
	}

	result := make(
		[]RankingItem,
		0,
		limit,
	)

	for i := 0; i < limit; i++ {
		container := containers[i]

		if container.Process == nil {
			continue
		}

		result = append(
			result,
			RankingItem{
				ContainerID:   container.ID,
				ContainerName: container.Name,
				PID:           container.Process.PID,
				Profile:       container.Profile,
				RAMPercent:    container.Process.MemoryPercent,
				CPUPercent:    container.Process.CPUPercent,
				VSZKB:         container.Process.VSZKB,
				RSSKB:         container.Process.RSSKB,
			},
		)
	}

	return result
}

/*
buildContainers crea el listado simple
de contenedores activos del proyecto.
*/
func buildContainers(
	analysis models.ContainerAnalysis,
) []ContainerSnapshot {
	result := make(
		[]ContainerSnapshot,
		0,
		len(analysis.ProjectContainers),
	)

	for _, container := range analysis.ProjectContainers {
		result = append(
			result,
			ContainerSnapshot{
				ID:      container.ID,
				Name:    container.Name,
				PID:     container.PID,
				Profile: container.Profile,
			},
		)
	}

	return result
}

/*
addSystemHistory agrega una muestra histórica.

Utilizamos un Stream de Valkey para conservar
las muestras a lo largo del tiempo.
*/
func (store *Store) addSystemHistory(
	snapshot SystemSnapshot,
) error {
	args := []string{
		"*",

		"timestamp",
		snapshot.Timestamp,

		"timestamp_unix",
		strconv.FormatInt(
			snapshot.TimestampUnix,
			10,
		),

		"total_ram_kb",
		strconv.FormatUint(
			snapshot.Memory.TotalKB,
			10,
		),

		"free_ram_kb",
		strconv.FormatUint(
			snapshot.Memory.FreeKB,
			10,
		),

		"used_ram_kb",
		strconv.FormatUint(
			snapshot.Memory.UsedKB,
			10,
		),

		"process_count",
		strconv.Itoa(
			snapshot.ProcessCount,
		),

		"project_containers",
		strconv.Itoa(
			snapshot.ProjectContainers,
		),

		"low_containers",
		strconv.Itoa(
			snapshot.LowContainers,
		),

		"high_containers",
		strconv.Itoa(
			snapshot.HighContainers,
		),

		"intruders",
		strconv.Itoa(
			snapshot.Intruders,
		),
	}

	command := store.client.B().
		Arbitrary("XADD").
		Keys(KeySystemHistory).
		Args(args...).
		Build()

	_, err := store.client.Do(
		store.ctx,
		command,
	).ToString()

	if err != nil {
		return fmt.Errorf(
			"no se pudo agregar historico en Valkey: %w",
			err,
		)
	}

	return nil
}

/*
SaveCycle almacena toda la telemetría necesaria
después del análisis de un ciclo del Daemon.
*/
func (store *Store) SaveCycle(
	telemetry *models.Telemetry,
	before models.ContainerAnalysis,
	after models.ContainerAnalysis,
	management models.ManagementResult,
) error {
	now := time.Now()

	systemSnapshot := SystemSnapshot{
		Timestamp:     now.Format(time.RFC3339),
		TimestampUnix: now.Unix(),

		Memory:       telemetry.Memory,
		ProcessCount: telemetry.ProcessCount,

		ProjectContainers: len(after.ProjectContainers),

		LowContainers: len(after.Low),

		HighContainers: len(after.High),

		Intruders: len(after.Intruders),
	}

	if err := store.setJSON(
		KeySystemCurrent,
		systemSnapshot,
	); err != nil {
		return err
	}

	if err := store.addSystemHistory(
		systemSnapshot,
	); err != nil {
		return err
	}

	topRAM := buildRanking(
		before.Rankings.ByRAM,
	)

	topCPU := buildRanking(
		before.Rankings.ByCPU,
	)

	if err := store.setJSON(
		KeyTopRAMCurrent,
		topRAM,
	); err != nil {
		return err
	}

	if err := store.setJSON(
		KeyTopCPUCurrent,
		topCPU,
	); err != nil {
		return err
	}

	if err := store.setJSON(
		KeyContainersCurrent,
		buildContainers(after),
	); err != nil {
		return err
	}

	managementSnapshot :=
		ManagementSnapshot{
			Timestamp: now.Format(time.RFC3339),

			Created: len(management.Created),

			/*
				Solo registramos que hubo una solicitud
				de eliminación.

				NO se registra todavía como una
				eliminación confirmada.
			*/
			RemovalRequested: len(management.Removed),

			FailedActions: len(management.Failed),
		}

	if err := store.setJSON(
		KeyManagementCurrent,
		managementSnapshot,
	); err != nil {
		return err
	}

	return nil
}

/*
SaveConfirmedDeletion registra una eliminación
confirmada mediante eBPF.

Se almacenan tres elementos:

1. Stream histórico de eventos.
2. Contador acumulado.
3. Último evento confirmado.
*/
func (store *Store) SaveConfirmedDeletion(
	event DeletedEvent,
) error {

	/*
		Primero agregamos el evento al histórico.
	*/
	args := []string{
		"*",

		"timestamp",
		event.Timestamp,

		"timestamp_unix",
		strconv.FormatInt(
			event.TimestampUnix,
			10,
		),

		"container_id",
		event.ContainerID,

		"container_name",
		event.ContainerName,

		"profile",
		event.Profile,

		"target_pid",
		strconv.Itoa(
			event.TargetPID,
		),

		"sender_pid",
		strconv.FormatUint(
			uint64(event.SenderPID),
			10,
		),

		"sender_tgid",
		strconv.FormatUint(
			uint64(event.SenderTGID),
			10,
		),

		"signal",
		strconv.Itoa(
			int(event.Signal),
		),

		"signal_name",
		event.SignalName,

		"sender_comm",
		event.SenderComm,

		"reason",
		event.Reason,
	}

	command :=
		store.client.B().
			Arbitrary("XADD").
			Keys(KeyDeletedEvents).
			Args(args...).
			Build()

	_, err :=
		store.client.Do(
			store.ctx,
			command,
		).ToString()

	if err != nil {
		return fmt.Errorf(
			"no se pudo guardar evento eBPF en Valkey: %w",
			err,
		)
	}

	/*
		Incrementamos el contador que posteriormente
		utilizará el panel de Grafana.
	*/
	counterCommand :=
		store.client.B().
			Arbitrary("INCR").
			Keys(KeyDeletedCount).
			Build()

	err = store.client.Do(
		store.ctx,
		counterCommand,
	).Error()

	if err != nil {
		return fmt.Errorf(
			"no se pudo incrementar contador eBPF: %w",
			err,
		)
	}

	/*
		Guardamos también el último evento.
	*/
	if err := store.setJSON(
		KeyDeletedCurrent,
		event,
	); err != nil {
		return err
	}

	return nil
}
