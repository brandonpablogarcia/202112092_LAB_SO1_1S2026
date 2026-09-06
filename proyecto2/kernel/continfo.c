#include <linux/init.h>
#include <linux/kernel.h>
#include <linux/module.h>
#include <linux/mm.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sysinfo.h>

#include <linux/sched.h>
#include <linux/sched/signal.h>
#include <linux/sched/mm.h>

#include <linux/hashtable.h>
#include <linux/slab.h>
#include <linux/ktime.h>
#include <linux/math64.h>
#include <linux/mutex.h>
#include <linux/string.h>

/*
 * Proyecto 2 - Sistemas Operativos 1
 * Carnet: 202112092
 *
 * Módulo encargado de exponer métricas de memoria
 * y procesos mediante /proc.
 */

#define PROC_NAME "continfo_pr2_so1_202112092"
#define CMDLINE_LEN 512

static struct proc_dir_entry *proc_entry;

/*
 * ============================================================
 * ESTRUCTURA PARA CALCULAR CPU DE FORMA DIFERENCIAL
 * ============================================================
 *
 * Guardamos el tiempo de CPU de cada PID en la lectura anterior.
 * Cuando /proc vuelve a ser leído, calculamos cuánto aumentó.
 */

struct cpu_sample {
    pid_t pid;
    u64 runtime_ns;
    u64 timestamp_ns;
    struct hlist_node node;
};

DEFINE_HASHTABLE(cpu_samples, 8);
static DEFINE_MUTEX(cpu_samples_mutex);

/*
 * Reemplaza caracteres que podrían romper el JSON.
 */
static void sanitize_text(char *text)
{
    int i;

    if (text == NULL)
        return;

    for (i = 0; text[i] != '\0'; i++) {
        if (text[i] == '"' ||
            text[i] == '\\' ||
            text[i] == '\n' ||
            text[i] == '\r' ||
            text[i] == '\t') {
            text[i] = ' ';
        }
    }
}

/*
 * Obtiene la línea de comandos de un proceso.
 *
 * No utilizamos get_cmdline() porque esa función
 * no está exportada para módulos externos en este Kernel.
 *
 * En su lugar obtenemos arg_start y arg_end desde mm_struct
 * y leemos esa región del espacio de memoria del proceso
 * mediante access_process_vm().
 */
static void get_process_cmdline(
    struct task_struct *task,
    char *buffer,
    size_t buffer_size,
    const char *fallback
)
{
    struct mm_struct *mm;

    unsigned long arg_start;
    unsigned long arg_end;

    size_t requested_len;

    int copied;
    int i;

    memset(buffer, 0, buffer_size);

    /*
     * Los kernel threads normalmente no poseen mm_struct.
     */
    mm = get_task_mm(task);

    if (mm == NULL) {
        snprintf(
            buffer,
            buffer_size,
            "[%s]",
            fallback
        );

        sanitize_text(buffer);
        return;
    }

    /*
     * Guardamos la región donde se encuentran
     * los argumentos del proceso.
     */
    arg_start = READ_ONCE(mm->arg_start);
    arg_end = READ_ONCE(mm->arg_end);

    if (arg_end <= arg_start) {
        mmput(mm);

        snprintf(
            buffer,
            buffer_size,
            "[%s]",
            fallback
        );

        sanitize_text(buffer);
        return;
    }

    /*
     * Nunca leeremos más de nuestro buffer.
     */
    requested_len = arg_end - arg_start;

    if (requested_len > buffer_size - 1)
        requested_len = buffer_size - 1;

    /*
     * access_process_vm está exportada para módulos GPL.
     */
    copied = access_process_vm(
        task,
        arg_start,
        buffer,
        (int)requested_len,
        0
    );

    mmput(mm);

    if (copied <= 0) {
        snprintf(
            buffer,
            buffer_size,
            "[%s]",
            fallback
        );

        sanitize_text(buffer);
        return;
    }

    if (copied >= buffer_size)
        copied = buffer_size - 1;

    /*
     * Los argumentos están separados por bytes NULL.
     * Los convertimos a espacios.
     */
    for (i = 0; i < copied; i++) {
        if (buffer[i] == '\0')
            buffer[i] = ' ';
    }

    buffer[copied] = '\0';

    sanitize_text(buffer);
}

/*
 * Calcula el porcentaje de CPU mediante diferencia
 * entre dos lecturas consecutivas.
 *
 * El resultado se devuelve multiplicado por 100:
 *
 * 10000 = 100.00 %
 *  5025 =  50.25 %
 */
static u64 calculate_cpu_percent(
    pid_t pid,
    u64 current_runtime_ns,
    u64 current_timestamp_ns
)
{
    struct cpu_sample *sample;
    struct cpu_sample *new_sample;

    u64 delta_runtime;
    u64 delta_time;
    u64 cpu_scaled = 0;

    mutex_lock(&cpu_samples_mutex);

    hash_for_each_possible(
        cpu_samples,
        sample,
        node,
        (unsigned long)pid
    ) {
        if (sample->pid == pid) {

            if (current_runtime_ns >= sample->runtime_ns &&
                current_timestamp_ns > sample->timestamp_ns) {

                delta_runtime =
                    current_runtime_ns - sample->runtime_ns;

                delta_time =
                    current_timestamp_ns - sample->timestamp_ns;

                /*
                 * Convertimos a centésimas de porcentaje.
                 */
                if (delta_time > 0) {
                    cpu_scaled = div64_u64(
                        delta_runtime * 10000ULL,
                        delta_time
                    );
                }
            }

            /*
             * Actualizamos la muestra para la siguiente lectura.
             */
            sample->runtime_ns = current_runtime_ns;
            sample->timestamp_ns = current_timestamp_ns;

            mutex_unlock(&cpu_samples_mutex);

            return cpu_scaled;
        }
    }

    /*
     * Primera vez que encontramos este PID.
     * Guardamos la muestra inicial.
     */
    new_sample = kmalloc(
        sizeof(struct cpu_sample),
        GFP_KERNEL
    );

    if (new_sample != NULL) {

        new_sample->pid = pid;
        new_sample->runtime_ns = current_runtime_ns;
        new_sample->timestamp_ns = current_timestamp_ns;

        hash_add(
            cpu_samples,
            &new_sample->node,
            (unsigned long)pid
        );
    }

    mutex_unlock(&cpu_samples_mutex);

    /*
     * En la primera lectura todavía no existe
     * información suficiente para calcular CPU.
     */
    return 0;
}

/*
 * Libera las muestras de CPU almacenadas.
 */
static void free_cpu_samples(void)
{
    struct cpu_sample *sample;
    struct hlist_node *tmp;
    int bucket;

    mutex_lock(&cpu_samples_mutex);

    hash_for_each_safe(
        cpu_samples,
        bucket,
        tmp,
        sample,
        node
    ) {
        hash_del(&sample->node);
        kfree(sample);
    }

    mutex_unlock(&cpu_samples_mutex);
}

/*
 * ============================================================
 * LECTURA PRINCIPAL DE /proc
 * ============================================================
 */
static int continfo_show(struct seq_file *m, void *v)
{
    struct sysinfo info;
    struct task_struct *task;

    u64 total_ram_kb;
    u64 free_ram_kb;
    u64 used_ram_kb;

    unsigned int process_count = 0;
    bool first_process = true;

    /*
     * Obtiene información general de RAM.
     */
    si_meminfo(&info);

    total_ram_kb =
        div64_u64(
            (u64)info.totalram * info.mem_unit,
            1024
        );

    free_ram_kb =
        div64_u64(
            (u64)info.freeram * info.mem_unit,
            1024
        );

    used_ram_kb =
        total_ram_kb - free_ram_kb;

    /*
     * Inicio del JSON.
     */
    seq_printf(
        m,
        "{\n"
        "  \"carnet\": \"202112092\",\n"
        "  \"memory\": {\n"
        "    \"total_kb\": %llu,\n"
        "    \"free_kb\": %llu,\n"
        "    \"used_kb\": %llu\n"
        "  },\n"
        "  \"processes\": [\n",
        (unsigned long long)total_ram_kb,
        (unsigned long long)free_ram_kb,
        (unsigned long long)used_ram_kb
    );

    /*
     * ========================================================
     * RECORRIDO DE task_struct
     * ========================================================
     *
     * for_each_process recorre los procesos visibles
     * por el Kernel.
     */
    for_each_process(task) {

        pid_t pid;

        char comm[TASK_COMM_LEN];
        char command_line[CMDLINE_LEN];

        struct mm_struct *mm;

        u64 vsz_kb = 0;
        u64 rss_kb = 0;

        u64 memory_scaled = 0;
        u64 cpu_scaled = 0;

        u64 runtime_ns;
        u64 timestamp_ns;

        pid = task_pid_nr(task);

        if (pid <= 0)
            continue;

        /*
         * Nombre corto del proceso.
         */
        get_task_comm(comm, task);

        sanitize_text(comm);

        /*
         * Línea completa de comandos.
         */
        get_process_cmdline(
            task,
            command_line,
            sizeof(command_line),
            comm
        );

        /*
         * Obtenemos información de memoria.
         *
         * Los kernel threads normalmente tienen mm == NULL.
         * En ese caso VSZ y RSS permanecen en cero.
         */
        mm = get_task_mm(task);

        if (mm != NULL) {

            vsz_kb =
                div64_u64(
                    (u64)mm->total_vm * PAGE_SIZE,
                    1024
                );

            rss_kb =
                div64_u64(
                    (u64)get_mm_rss(mm) * PAGE_SIZE,
                    1024
                );

            mmput(mm);
        }

        /*
         * Porcentaje de memoria.
         *
         * Resultado multiplicado por 100:
         * 125 = 1.25 %
         */
        if (total_ram_kb > 0) {
            memory_scaled =
                div64_u64(
                    rss_kb * 10000ULL,
                    total_ram_kb
                );
        }

        /*
         * sum_exec_runtime representa el tiempo total
         * que el proceso ha utilizado CPU.
         */
        runtime_ns =
            READ_ONCE(task->se.sum_exec_runtime);

        timestamp_ns =
            ktime_get_ns();

        cpu_scaled =
            calculate_cpu_percent(
                pid,
                runtime_ns,
                timestamp_ns
            );

        /*
         * Se agrega coma entre objetos JSON,
         * excepto antes del primero.
         */
        if (!first_process)
            seq_puts(m, ",\n");

        first_process = false;

        seq_printf(
            m,
            "    {\n"
            "      \"pid\": %d,\n"
            "      \"name\": \"%s\",\n"
            "      \"command_line\": \"%s\",\n"
            "      \"vsz_kb\": %llu,\n"
            "      \"rss_kb\": %llu,\n"
            "      \"memory_percent\": %llu.%02llu,\n"
            "      \"cpu_percent\": %llu.%02llu\n"
            "    }",
            pid,
            comm,
            command_line,
            (unsigned long long)vsz_kb,
            (unsigned long long)rss_kb,

            (unsigned long long)(memory_scaled / 100),
            (unsigned long long)(memory_scaled % 100),

            (unsigned long long)(cpu_scaled / 100),
            (unsigned long long)(cpu_scaled % 100)
        );

        process_count++;
    }

    /*
     * Cerramos el arreglo y agregamos
     * el número total de procesos.
     */
    seq_printf(
        m,
        "\n"
        "  ],\n"
        "  \"process_count\": %u\n"
        "}\n",
        process_count
    );

    return 0;
}

static int continfo_open(
    struct inode *inode,
    struct file *file
)
{
    return single_open(
        file,
        continfo_show,
        NULL
    );
}

static const struct proc_ops continfo_ops = {
    .proc_open = continfo_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

/*
 * ============================================================
 * INICIALIZACIÓN
 * ============================================================
 */
static int __init continfo_init(void)
{
    hash_init(cpu_samples);

    proc_entry = proc_create(
        PROC_NAME,
        0444,
        NULL,
        &continfo_ops
    );

    if (proc_entry == NULL) {

        pr_err(
            "SO1 202112092: No se pudo crear /proc/%s\n",
            PROC_NAME
        );

        return -ENOMEM;
    }

    pr_info(
        "SO1 202112092: Modulo continfo cargado correctamente\n"
    );

    pr_info(
        "SO1 202112092: Archivo creado en /proc/%s\n",
        PROC_NAME
    );

    return 0;
}

/*
 * ============================================================
 * DESCARGA
 * ============================================================
 */
static void __exit continfo_exit(void)
{
    proc_remove(proc_entry);

    free_cpu_samples();

    pr_info(
        "SO1 202112092: Modulo continfo descargado correctamente\n"
    );
}

module_init(continfo_init);
module_exit(continfo_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("202112092");
MODULE_DESCRIPTION(
    "Proyecto 2 SO1 - Telemetria de procesos y contenedores"
);
MODULE_VERSION("2.0");