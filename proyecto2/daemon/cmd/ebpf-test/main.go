package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cilium/ebpf/ringbuf"

	"so1.local/daemon202112092/ebpfmonitor"
)

/*
signalName permite mostrar de forma amigable
las señales más importantes del proyecto.
*/
func signalName(signal int32) string {
	switch signal {

	case int32(syscall.SIGTERM):
		return "SIGTERM"

	case int32(syscall.SIGKILL):
		return "SIGKILL"

	case int32(syscall.SIGINT):
		return "SIGINT"

	case int32(syscall.SIGHUP):
		return "SIGHUP"

	default:
		return fmt.Sprintf(
			"SIGNAL_%d",
			signal,
		)
	}
}

/*
commString convierte char[16] proveniente
de eBPF en un string de Go.
*/
func commString(comm [16]byte) string {
	return strings.TrimRight(
		string(comm[:]),
		"\x00",
	)
}

func main() {

	if os.Geteuid() != 0 {
		fmt.Println(
			"[ERROR] Este monitor debe ejecutarse con sudo.",
		)

		os.Exit(1)
	}

	objectPath :=
		"../ebpf/kill_monitor.bpf.o"

	if len(os.Args) > 1 {
		objectPath = os.Args[1]
	}

	fmt.Println("====================================================")
	fmt.Println("MONITOR eBPF - PROYECTO 2 SO1")
	fmt.Println("Carnet: 202112092")
	fmt.Println("====================================================")

	fmt.Printf(
		"[INFO] Objeto eBPF: %s\n",
		objectPath,
	)

	monitor, err :=
		ebpfmonitor.New(
			objectPath,
		)

	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"[ERROR] %v\n",
			err,
		)

		os.Exit(1)
	}

	defer monitor.Close()

	fmt.Println(
		"[OK] Programa eBPF cargado correctamente.",
	)

	fmt.Println(
		"[OK] Adjuntado a syscalls:sys_enter_kill.",
	)

	fmt.Println(
		"[INFO] Esperando señales kill()...",
	)

	fmt.Println(
		"[INFO] Ctrl+C para finalizar.",
	)

	signalChannel :=
		make(chan os.Signal, 1)

	signal.Notify(
		signalChannel,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	/*
		Cuando recibamos Ctrl+C cerramos el
		monitor para desbloquear ReadEvent().
	*/
	go func() {
		<-signalChannel

		fmt.Println()
		fmt.Println(
			"[INFO] Finalizando monitor eBPF...",
		)

		monitor.Close()
	}()

	for {

		event, err :=
			monitor.ReadEvent()

		if err != nil {

			if errors.Is(
				err,
				ringbuf.ErrClosed,
			) {
				break
			}

			fmt.Fprintf(
				os.Stderr,
				"[ERROR] %v\n",
				err,
			)

			continue
		}

		fmt.Println()
		fmt.Println(
			"---------------- EVENTO eBPF ----------------",
		)

		fmt.Printf(
			"Proceso emisor: %s\n",
			commString(event.Comm),
		)

		fmt.Printf(
			"PID emisor:     %d\n",
			event.SenderPID,
		)

		fmt.Printf(
			"TGID emisor:    %d\n",
			event.SenderTGID,
		)

		fmt.Printf(
			"PID objetivo:   %d\n",
			event.TargetPID,
		)

		fmt.Printf(
			"Señal:          %d (%s)\n",
			event.Signal,
			signalName(event.Signal),
		)

		fmt.Println(
			"----------------------------------------------",
		)
	}

	fmt.Println(
		"[OK] Monitor eBPF finalizado correctamente.",
	)
}
