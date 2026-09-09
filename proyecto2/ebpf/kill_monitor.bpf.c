#include "vmlinux.h"

#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

/*
Proyecto 2 - Sistemas Operativos 1
Carnet: 202112092

Programa eBPF encargado de interceptar
la syscall kill().

En esta primera etapa únicamente capturamos
el evento y lo enviamos al espacio de usuario
mediante un Ring Buffer.
*/

char LICENSE[] SEC("license") = "GPL";

/*
Información enviada desde eBPF hacia Go.

La estructura ocupa 40 bytes:

8  timestamp
4  sender_tgid
4  sender_pid
4  target_pid
4  signal
16 comm
*/
struct kill_event {
    __u64 timestamp_ns;

    __u32 sender_tgid;
    __u32 sender_pid;

    __s32 target_pid;
    __s32 signal;

    char comm[16];
};

/*
Ring Buffer utilizado para enviar eventos
desde el Kernel hacia espacio de usuario.
*/
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 20);
} events SEC(".maps");

/*
Tracepoint:

syscalls:sys_enter_kill

Se ejecutará cada vez que algún proceso
realice la syscall kill().
*/
SEC("tracepoint/syscalls/sys_enter_kill")
int trace_kill(
    struct trace_event_raw_sys_enter *ctx
)
{
    struct kill_event *event;

    __u64 pid_tgid;

    /*
    Reservamos espacio dentro del Ring Buffer.
    */
    event = bpf_ringbuf_reserve(
        &events,
        sizeof(*event),
        0
    );

    if (!event)
        return 0;

    /*
    PID/TGID del proceso que ejecutó kill().
    */
    pid_tgid = bpf_get_current_pid_tgid();

    event->timestamp_ns =
        bpf_ktime_get_ns();

    event->sender_tgid =
        pid_tgid >> 32;

    event->sender_pid =
        (__u32)pid_tgid;

    /*
    kill(pid, signal)

    args[0] = PID destino
    args[1] = señal
    */
    event->target_pid =
        (__s32)ctx->args[0];

    event->signal =
        (__s32)ctx->args[1];

    /*
    Nombre del proceso que realizó kill().
    */
    bpf_get_current_comm(
        event->comm,
        sizeof(event->comm)
    );

    /*
    Enviamos el evento al programa Go.
    */
    bpf_ringbuf_submit(
        event,
        0
    );

    return 0;
}

/*
 * Tracepoint:
 *
 * signal:signal_generate
 *
 * Este evento se produce cuando el Kernel genera
 * realmente una señal hacia un proceso.
 *
 * Nos permite detectar terminaciones aunque Docker
 * no utilice directamente la syscall kill().
 */
SEC("tracepoint/signal/signal_generate")
int trace_signal_generate(
    struct trace_event_raw_signal_generate *ctx
)
{
    struct kill_event *event;

    __u64 pid_tgid;

    /*
     * Nos interesan principalmente las señales
     * utilizadas para finalizar procesos.
     */
    if (ctx->sig != 15 && ctx->sig != 9)
        return 0;

    event = bpf_ringbuf_reserve(
        &events,
        sizeof(*event),
        0
    );

    if (!event)
        return 0;

    pid_tgid =
        bpf_get_current_pid_tgid();

    event->timestamp_ns =
        bpf_ktime_get_ns();

    /*
     * El proceso actual corresponde al emisor
     * de la señal.
     */
    event->sender_tgid =
        pid_tgid >> 32;

    event->sender_pid =
        (__u32)pid_tgid;

    /*
     * signal_generate ya nos proporciona el PID
     * del proceso receptor.
     */
    event->target_pid =
        ctx->pid;

    event->signal =
        ctx->sig;

    bpf_get_current_comm(
        event->comm,
        sizeof(event->comm)
    );

    bpf_ringbuf_submit(
        event,
        0
    );

    return 0;
}