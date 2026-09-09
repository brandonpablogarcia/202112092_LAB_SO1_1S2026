package ebpfmonitor

import (
	"strings"
	"sync"
	"syscall"
	"time"
)

/*
PendingDeletion representa una eliminación que
el Daemon está a punto de ejecutar.

El PID se registra ANTES de docker stop para evitar
perder el evento eBPF si la señal ocurre rápidamente.
*/
type PendingDeletion struct {
	ContainerID   string
	ContainerName string
	Profile       string
	PID           int
	Reason        string
	RegisteredAt  time.Time
}

/*
Confirmation combina:

- La acción esperada por el Daemon.
- El evento real capturado por eBPF.
*/
type Confirmation struct {
	Pending     PendingDeletion
	Event       Event
	ConfirmedAt time.Time
}

type pendingEntry struct {
	Pending      PendingDeletion
	Confirmation chan Confirmation
}

/*
Tracker mantiene los PID que el Daemon
espera observar mediante eBPF.
*/
type Tracker struct {
	mu      sync.Mutex
	pending map[int]*pendingEntry
}

/*
NewTracker crea el registro de eliminaciones pendientes.
*/
func NewTracker() *Tracker {
	return &Tracker{
		pending: make(
			map[int]*pendingEntry,
		),
	}
}

/*
Expect registra un PID antes de ejecutar docker stop.
*/
func (tracker *Tracker) Expect(
	pending PendingDeletion,
) {
	if pending.PID <= 0 {
		return
	}

	if pending.RegisteredAt.IsZero() {
		pending.RegisteredAt = time.Now()
	}

	entry := &pendingEntry{
		Pending: pending,
		Confirmation: make(
			chan Confirmation,
			1,
		),
	}

	tracker.mu.Lock()

	tracker.pending[pending.PID] = entry

	tracker.mu.Unlock()
}

/*
Cancel elimina una espera que ya no necesitamos.

Por ejemplo, si docker stop falló.
*/
func (tracker *Tracker) Cancel(pid int) {
	tracker.mu.Lock()

	delete(
		tracker.pending,
		pid,
	)

	tracker.mu.Unlock()
}

/*
Match recibe un evento proveniente del Kernel.

Solamente acepta:

- SIGTERM
- SIGKILL

y únicamente cuando el PID objetivo pertenece
a una eliminación esperada por el Daemon.
*/
func (tracker *Tracker) Match(
	event Event,
) bool {
	if event.TargetPID <= 0 {
		return false
	}

	if event.Signal != int32(syscall.SIGTERM) &&
		event.Signal != int32(syscall.SIGKILL) {

		return false
	}

	pid := int(event.TargetPID)

	tracker.mu.Lock()

	entry, exists :=
		tracker.pending[pid]

	tracker.mu.Unlock()

	if !exists {
		return false
	}

	confirmation := Confirmation{
		Pending: entry.Pending,
		Event:   event,

		ConfirmedAt: time.Now(),
	}

	/*
		Canal con buffer 1.

		Si Docker llegara a producir más de una señal
		para el mismo PID, conservamos la primera
		confirmación relevante.
	*/
	select {

	case entry.Confirmation <- confirmation:
		return true

	default:
		/*
			Ya existe una confirmación pendiente
			para este PID.

			Ignoramos eventos duplicados.
		*/
		return false
	}
}

/*
Wait espera durante un tiempo limitado a que
eBPF confirme la señal correspondiente al PID.
*/
func (tracker *Tracker) Wait(
	pid int,
	timeout time.Duration,
) (
	Confirmation,
	bool,
) {
	tracker.mu.Lock()

	entry, exists :=
		tracker.pending[pid]

	tracker.mu.Unlock()

	if !exists {
		return Confirmation{}, false
	}

	timer :=
		time.NewTimer(timeout)

	defer timer.Stop()

	select {

	case confirmation :=
		<-entry.Confirmation:

		tracker.Cancel(pid)

		return confirmation, true

	case <-timer.C:

		tracker.Cancel(pid)

		return Confirmation{}, false
	}
}

/*
CleanupExpired elimina registros viejos para
evitar mantener PID obsoletos.
*/
func (tracker *Tracker) CleanupExpired(
	maxAge time.Duration,
) {
	now := time.Now()

	tracker.mu.Lock()

	for pid, entry := range tracker.pending {

		if now.Sub(
			entry.Pending.RegisteredAt,
		) > maxAge {

			delete(
				tracker.pending,
				pid,
			)
		}
	}

	tracker.mu.Unlock()
}

/*
SignalName convierte el número de señal
en un nombre legible.
*/
func SignalName(signal int32) string {
	switch signal {

	case int32(syscall.SIGTERM):
		return "SIGTERM"

	case int32(syscall.SIGKILL):
		return "SIGKILL"

	case int32(syscall.SIGINT):
		return "SIGINT"

	default:
		return "SIGNAL"
	}
}

/*
CommString convierte char[16] proveniente
del programa C eBPF a string de Go.
*/
func CommString(comm [16]byte) string {
	return strings.TrimRight(
		string(comm[:]),
		"\x00",
	)
}
