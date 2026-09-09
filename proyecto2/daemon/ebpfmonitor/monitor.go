package ebpfmonitor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

/*
Event debe coincidir exactamente con la estructura
struct kill_event definida dentro del programa C eBPF.

Estructura C esperada:

	struct kill_event {
	    __u64 timestamp_ns;

	    __u32 sender_tgid;
	    __u32 sender_pid;

	    __s32 target_pid;
	    __s32 signal;

	    char comm[16];
	};
*/
type Event struct {
	TimestampNS uint64

	SenderTGID uint32
	SenderPID  uint32

	TargetPID int32
	Signal    int32

	Comm [16]byte
}

/*
Monitor mantiene todos los recursos utilizados
por el subsistema eBPF.

Tenemos dos tracepoints:

 1. syscalls:sys_enter_kill
    Detecta llamadas directas a kill().

 2. signal:signal_generate
    Detecta la generación efectiva de SIGTERM/SIGKILL.

El segundo es especialmente útil para Docker,
containerd y otros runtimes.
*/
type Monitor struct {
	collection *ebpf.Collection

	killTraceLink   link.Link
	signalTraceLink link.Link

	reader *ringbuf.Reader

	closeOnce sync.Once
}

/*
New carga el archivo .bpf.o, localiza los programas
y mapas eBPF, y conecta los dos tracepoints.
*/
func New(
	objectPath string,
) (*Monitor, error) {

	/*
		Elimina la limitación de memoria bloqueada
		para permitir cargar mapas y programas eBPF.
	*/
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf(
			"no se pudo ajustar RLIMIT_MEMLOCK: %w",
			err,
		)
	}

	/*
		Leemos las especificaciones contenidas
		dentro del objeto ELF eBPF.
	*/
	spec, err := ebpf.LoadCollectionSpec(
		objectPath,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo leer el objeto eBPF %s: %w",
			objectPath,
			err,
		)
	}

	/*
		Cargamos programas y mapas en el Kernel.
	*/
	collection, err := ebpf.NewCollection(
		spec,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"el Kernel rechazo el programa eBPF: %w",
			err,
		)
	}

	/*
		------------------------------------------------
		PROGRAMA 1:
		syscalls:sys_enter_kill
		------------------------------------------------
	*/
	killProgram, exists :=
		collection.Programs["trace_kill"]

	if !exists {
		collection.Close()

		return nil, fmt.Errorf(
			"no existe el programa trace_kill dentro del objeto eBPF",
		)
	}

	/*
		------------------------------------------------
		PROGRAMA 2:
		signal:signal_generate
		------------------------------------------------
	*/
	signalProgram, exists :=
		collection.Programs["trace_signal_generate"]

	if !exists {
		collection.Close()

		return nil, fmt.Errorf(
			"no existe el programa trace_signal_generate dentro del objeto eBPF",
		)
	}

	/*
		------------------------------------------------
		MAPA RING BUFFER
		------------------------------------------------
	*/
	eventMap, exists :=
		collection.Maps["events"]

	if !exists {
		collection.Close()

		return nil, fmt.Errorf(
			"no existe el mapa events dentro del objeto eBPF",
		)
	}

	/*
		------------------------------------------------
		ADJUNTAR sys_enter_kill
		------------------------------------------------
	*/
	killTraceLink, err := link.Tracepoint(
		"syscalls",
		"sys_enter_kill",
		killProgram,
		nil,
	)

	if err != nil {
		collection.Close()

		return nil, fmt.Errorf(
			"no se pudo adjuntar sys_enter_kill: %w",
			err,
		)
	}

	/*
		------------------------------------------------
		ADJUNTAR signal_generate
		------------------------------------------------
	*/
	signalTraceLink, err := link.Tracepoint(
		"signal",
		"signal_generate",
		signalProgram,
		nil,
	)

	if err != nil {

		_ = killTraceLink.Close()

		collection.Close()

		return nil, fmt.Errorf(
			"no se pudo adjuntar signal_generate: %w",
			err,
		)
	}

	/*
		------------------------------------------------
		RING BUFFER
		------------------------------------------------

		El mismo mapa "events" recibe información
		proveniente de ambos programas eBPF.
	*/
	reader, err := ringbuf.NewReader(
		eventMap,
	)

	if err != nil {

		_ = signalTraceLink.Close()
		_ = killTraceLink.Close()

		collection.Close()

		return nil, fmt.Errorf(
			"no se pudo abrir el Ring Buffer: %w",
			err,
		)
	}

	/*
		Todo fue cargado correctamente.
	*/
	monitor := &Monitor{
		collection: collection,

		killTraceLink: killTraceLink,

		signalTraceLink: signalTraceLink,

		reader: reader,
	}

	return monitor, nil
}

/*
ReadEvent espera un registro enviado por alguno
de los programas eBPF y lo convierte a Event.
*/
func (monitor *Monitor) ReadEvent() (
	Event,
	error,
) {

	record, err :=
		monitor.reader.Read()

	if err != nil {
		return Event{}, err
	}

	var event Event

	/*
		El programa eBPF escribe los bytes de
		struct kill_event directamente al Ring Buffer.

		Los convertimos a la estructura Event de Go.
	*/
	err = binary.Read(
		bytes.NewReader(
			record.RawSample,
		),
		binary.LittleEndian,
		&event,
	)

	if err != nil {
		return Event{}, fmt.Errorf(
			"no se pudo decodificar evento eBPF: %w",
			err,
		)
	}

	return event, nil
}

/*
Close libera todos los recursos de eBPF.

El orden utilizado es:

1. Cerrar Ring Buffer.
2. Desconectar signal_generate.
3. Desconectar sys_enter_kill.
4. Cerrar colección eBPF.

sync.Once evita ejecutar la limpieza dos veces.
*/
func (monitor *Monitor) Close() {

	monitor.closeOnce.Do(
		func() {

			if monitor.reader != nil {
				_ = monitor.reader.Close()
			}

			if monitor.signalTraceLink != nil {
				_ = monitor.signalTraceLink.Close()
			}

			if monitor.killTraceLink != nil {
				_ = monitor.killTraceLink.Close()
			}

			if monitor.collection != nil {
				monitor.collection.Close()
			}
		},
	)
}
