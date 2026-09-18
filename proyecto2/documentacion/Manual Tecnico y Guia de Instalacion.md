UNIVESIDAD DE SAN CARLOS DE GUATEMALA

FACULTAD DE INGENIERIA

ESCUELA DE CIENCAS Y SISTEMAS

LABORATORIO SISTEMAS OPERATIVOS 1 

SECCIÓN P

SEGUNDO SEMESTRE 2026

AUX. JOSÉ DANIEL LORENZANA MEDINA




<p align="center"> MANUAL TECNICO Y GUIA DE INSTALACION </p>



BRANDON EDUARDO PABLO GARCIA

202112092

Guatemala


---

# INTRODUCCIÓN

El presente manual técnico describe la arquitectura, instalación, configuración, compilación, ejecución, pruebas y mantenimiento del Proyecto 2 de Sistemas Operativos 1.

El proyecto implementa un sistema capaz de generar, monitorear, analizar y administrar contenedores Docker de forma automática. Para lograrlo se integran diferentes tecnologías de bajo nivel y observabilidad: un módulo del Kernel desarrollado en C, un Daemon desarrollado en Go, programas eBPF para confirmar terminaciones de procesos, Valkey como almacenamiento de telemetría y Grafana para la visualización de métricas.

El sistema obtiene información directamente del Kernel de Linux mediante un archivo `/proc`, relaciona los procesos con los contenedores Docker activos, clasifica los contenedores según su perfil y consumo, mantiene una cantidad mínima de contenedores de bajo y alto consumo, elimina candidatos cuando es necesario y registra únicamente como confirmadas aquellas eliminaciones que son observadas por eBPF a nivel del Kernel.

La información histórica y actual se almacena en Valkey y se presenta por medio de un dashboard de Grafana.

---

# OBJETIVOS

## Objetivo general

- Implementar un sistema de administración y monitoreo automático de contenedores Docker utilizando mecanismos del Kernel de Linux, eBPF, Go, Valkey y Grafana.

## Objetivos específicos

- Obtener información de memoria y procesos directamente desde el Kernel de Linux.
- Exponer la telemetría del sistema mediante un archivo `/proc`.
- Relacionar procesos del sistema con los contenedores Docker activos.
- Clasificar los contenedores según su perfil de consumo.
- Mantener como mínimo 3 contenedores de bajo consumo y 2 contenedores de alto consumo.
- Generar contenedores automáticamente mediante un Cronjob.
- Eliminar contenedores candidatos de acuerdo con criterios de RAM, VSZ, RSS y CPU.
- Confirmar eliminaciones reales mediante eBPF.
- Almacenar telemetría actual e histórica en Valkey.
- Visualizar métricas en Grafana.
- Garantizar una limpieza controlada al finalizar el Daemon.

---

# TECNOLOGÍAS UTILIZADAS

| Tecnología | Uso dentro del proyecto |
|---|---|
| C | Desarrollo del módulo Kernel y programa eBPF |
| Go | Desarrollo del Daemon y servicios de administración |
| Docker | Creación y ejecución de contenedores |
| Docker Compose | Orquestación de Valkey y Grafana |
| Linux Kernel Modules | Obtención de información del sistema |
| `/proc` | Exposición de telemetría del Kernel |
| eBPF | Intercepción y confirmación de señales de terminación |
| Clang/LLVM | Compilación del código eBPF |
| bpftool | Inspección y soporte para BTF/eBPF |
| Valkey | Almacenamiento de métricas y eventos |
| Grafana | Visualización de métricas |
| Bash | Scripts de compilación, carga y automatización |
| Cron | Generación periódica de contenedores |

> **Importante:** el proyecto utiliza la imagen oficial de Valkey:
>
> `valkey/valkey:latest`
>
> No se debe sustituir por Redis, debido a la restricción establecida para el proyecto.

---

# ARQUITECTURA GENERAL

```text
Cronjob
   |
   v
Generador de contenedores
   |
   v
Docker
   |
   +-------------------------------+
   |                               |
   v                               v
Módulo Kernel                  Daemon Go
   |                               |
   v                               |
/proc/continfo_pr2_so1_202112092   |
   |                               |
   +--------------->---------------+
                                   |
                                   v
                         Análisis de contenedores
                                   |
                                   v
                           Gestión automática
                                   |
                                   v
                              docker stop
                                   |
                                   v
                                 eBPF
                                   |
                              Ring Buffer
                                   |
                                   v
                              Daemon Go
                                   |
                                   v
                                Valkey
                                   |
                                   v
                                Grafana
```

El flujo principal es el siguiente:

1. El Cronjob ejecuta un script que genera contenedores.
2. El módulo Kernel obtiene información de procesos y memoria.
3. El archivo `/proc` expone la telemetría en formato JSON.
4. El Daemon lee `/proc` y consulta Docker.
5. Los PID de Docker se relacionan con los procesos reportados por el Kernel.
6. Los contenedores se clasifican y ordenan según su consumo.
7. El Daemon crea o elimina contenedores según las restricciones.
8. eBPF observa la señal de terminación.
9. La eliminación se confirma únicamente si el PID observado coincide con una eliminación esperada.
10. Valkey almacena telemetría y eventos.
11. Grafana visualiza los datos.

---

# ESTRUCTURA DEL PROYECTO

```text
proyecto2/
├── containers/
│   ├── high_cpu/
│   │   └── Dockerfile
│   ├── high_ram/
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── main.go
│   ├── intruder/
│   │   ├── Dockerfile
│   │   └── attack.sh
│   ├── low/
│   │   └── Dockerfile
│   └── scripts/
│       └── create_containers.sh
│
├── cron/
│   ├── install_cron.sh
│   └── remove_cron.sh
│
├── daemon/
│   ├── ebpfmonitor/
│   │   ├── listener.go
│   │   ├── monitor.go
│   │   └── tracker.go
│   ├── models/
│   │   ├── container.go
│   │   ├── decision.go
│   │   ├── management.go
│   │   └── telemetry.go
│   ├── services/
│   │   ├── container_analyzer.go
│   │   ├── container_manager.go
│   │   ├── docker_service.go
│   │   ├── proc_reader.go
│   │   └── system_manager.go
│   ├── valkey/
│   │   └── store.go
│   ├── go.mod
│   ├── go.sum
│   └── main.go
│
├── ebpf/
│   ├── build.sh
│   └── kill_monitor.bpf.c
│
├── kernel/
│   ├── continfo.c
│   ├── Makefile
│   ├── load_module.sh
│   └── unload_module.sh
│
├── monitoring/
│   ├── docker-compose.yml
│   └── grafana/
│       ├── dashboards/
│       │   └── dashboard-202112092.json
│       └── provisioning/
│           └── datasources/
│               └── valkey.yml
│
└── README.md
```

---

# REQUISITOS DEL SISTEMA

El proyecto fue desarrollado y probado en Linux.

## Herramientas requeridas

```bash
docker --version
docker compose version
go version
clang --version
bpftool version
make --version
```

## Dependencias recomendadas en Ubuntu

```bash
sudo apt update

sudo apt install -y \
    build-essential \
    linux-headers-$(uname -r) \
    clang \
    llvm \
    bpftool \
    libbpf-dev \
    jq \
    mokutil
```

## Secure Boot

Para cargar módulos Kernel no firmados puede ser necesario deshabilitar Secure Boot.

Verificación:

```bash
mokutil --sb-state
```

Si aparece:

```text
SecureBoot enabled
```

se debe ingresar al BIOS/UEFI y deshabilitar Secure Boot antes de cargar el módulo.



---

# CONSTRUCCIÓN DE LAS IMÁGENES DOCKER

Ubicarse en:

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2
```

## Contenedor LOW

```bash
docker build \
-t so1-202112092-low \
./containers/low
```

## Contenedor HIGH RAM

```bash
docker build \
-t so1-202112092-high-ram \
./containers/high_ram
```

## Contenedor HIGH CPU

```bash
docker build \
-t so1-202112092-high-cpu \
./containers/high_cpu
```

## Contenedor INTRUDER

```bash
docker build \
-t so1-202112092-intruder \
./containers/intruder
```

## Verificación

```bash
docker images | grep so1-202112092
```

---

# MONITORING: VALKEY Y GRAFANA

Ubicarse en:

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/monitoring
```

Levantar servicios:

```bash
docker compose up -d
```

Verificar estado:

```bash
docker compose ps
```

Valkey debe responder:

```bash
docker exec valkey-so1-202112092 valkey-cli ping
```

Respuesta esperada:

```text
PONG
```

Grafana queda disponible en:

```text
http://localhost:3000
```

El datasource utilizado es:

```text
Valkey-SO1-202112092
```

---

# MÓDULO KERNEL

El módulo Kernel se encuentra en:

```text
proyecto2/kernel/
```

## Compilación

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/kernel
make
```

## Carga del módulo

```bash
./load_module.sh
```

Verificar:

```bash
lsmod | grep continfo
```

## Archivo `/proc`

El módulo genera:

```text
/proc/continfo_pr2_so1_202112092
```

Consultar:

```bash
cat /proc/continfo_pr2_so1_202112092 | jq
```

El JSON contiene información como:

- memoria RAM total;
- memoria RAM libre;
- memoria RAM utilizada;
- cantidad de procesos;
- PID;
- nombre del proceso;
- VSZ;
- RSS;
- porcentaje de memoria;
- porcentaje de CPU.

## Descarga del módulo

```bash
./unload_module.sh
```

---

# GENERACIÓN AUTOMÁTICA DE CONTENEDORES

El script principal es:

```text
containers/scripts/create_containers.sh
```

Su función es generar cargas simuladas de forma aleatoria.

Los perfiles manejados son:

```text
low
high_ram
high_cpu
intruder
```

El Cronjob se instala mediante:

```text
cron/install_cron.sh
```

Y se elimina mediante:

```text
cron/remove_cron.sh
```

La periodicidad utilizada es cada 2 minutos.

Verificación:

```bash
sudo crontab -l
```

---

# DAEMON EN GO

El Daemon representa el núcleo de coordinación del proyecto.

Ubicación:

```text
proyecto2/daemon/
```

## Compilación

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/daemon

go mod tidy

go build -o daemon-so1 .
```

## Ejecución

```bash
sudo ./daemon-so1
```

Se requieren privilegios de root porque el Daemon interactúa con el módulo Kernel y eBPF.

## Ciclo del Daemon

El Daemon ejecuta un ciclo aproximadamente cada 30 segundos.

Durante cada ciclo realiza:

1. lectura de `/proc`;
2. consulta de Docker;
3. relación Docker PID ↔ Kernel PID;
4. clasificación de contenedores;
5. generación de rankings;
6. corrección de mínimos;
7. selección de candidatos;
8. eliminación de candidatos;
9. espera de confirmación eBPF;
10. almacenamiento en Valkey.

---

# RELACIÓN DOCKER ↔ PID ↔ KERNEL

Docker permite obtener el PID principal de cada contenedor mediante `docker inspect`.

El Daemon obtiene esos PID y los compara con la telemetría expuesta por el módulo Kernel.

Flujo:

```text
Docker Container
      |
      v
docker inspect
      |
      v
PID principal
      |
      v
/proc
      |
      v
ProcessInfo
```

Esto permite disponer para cada contenedor de:

- PID;
- RAM;
- VSZ;
- RSS;
- CPU.

---

# CLASIFICACIÓN DE CONTENEDORES

La clasificación utilizada es:

```text
LOW
HIGH_RAM
HIGH_CPU
INTRUDER
UNKNOWN
```

Para efectos de restricciones:

```text
HIGH = HIGH_RAM + HIGH_CPU
```

El sistema mantiene como mínimo:

```text
LOW  >= 3
HIGH >= 2
```

Grafana y Valkey se consideran infraestructura protegida y nunca deben ser eliminados.

---

# MÉTRICAS Y RANKINGS

Para seleccionar candidatos se utilizan cuatro métricas:

- porcentaje de RAM;
- VSZ;
- RSS;
- porcentaje de CPU.

El Daemon genera rankings independientes por cada métrica y calcula un puntaje combinado.

Los candidatos pueden corresponder a:

- contenedores intrusos;
- exceso de LOW;
- exceso de HIGH.

---

# GESTIÓN AUTÓNOMA DE CONTENEDORES

El Daemon puede crear contenedores faltantes y eliminar contenedores excedentes.

Ejemplo:

```text
LOW = 2
HIGH = 3
INTRUDER = 1
```

Acciones esperadas:

```text
Crear 1 LOW
Eliminar 1 HIGH
Eliminar 1 INTRUDER
```

Resultado esperado:

```text
LOW = 3
HIGH = 2
INTRUDER = 0
```

# VALKEY

Valkey almacena el estado actual y el histórico del sistema.

## Claves principales

```text
so1:202112092:system:current
so1:202112092:system:history

so1:202112092:top:ram:current
so1:202112092:top:cpu:current

so1:202112092:top:ram:historical
so1:202112092:top:cpu:historical

so1:202112092:containers:current
so1:202112092:management:current

so1:202112092:events:deleted
so1:202112092:events:deleted:count
so1:202112092:events:deleted:current
```

## Consultar claves

```bash
docker exec valkey-so1-202112092 \
valkey-cli --scan \
--pattern 'so1:202112092:*'
```

## Consultar telemetría actual

```bash
docker exec valkey-so1-202112092 \
valkey-cli --raw \
GET so1:202112092:system:current \
| jq
```
---

# eBPF

El subsistema eBPF confirma que una terminación realmente ocurrió a nivel del Kernel.

Ubicación:

```text
proyecto2/ebpf/
```

## Compilación

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/ebpf
./build.sh
```

El script genera:

```text
vmlinux.h
kill_monitor.bpf.o
```

## Tracepoints utilizados

```text
syscalls:sys_enter_kill
signal:signal_generate
```

El primero permite observar llamadas directas a `kill()`.

El segundo permite observar la generación efectiva de señales como SIGTERM y SIGKILL, incluso cuando Docker o containerd utilizan mecanismos diferentes.



---

# COMUNICACIÓN eBPF → GO

Los eventos eBPF se envían al espacio de usuario mediante un **Ring Buffer**.

Cada evento contiene información como:

- PID emisor;
- TGID emisor;
- PID objetivo;
- señal;
- proceso emisor;
- timestamp.

El Daemon mantiene un registro de eliminaciones pendientes.

Cuando eBPF detecta una señal:

```text
Evento eBPF
   |
   v
¿PID objetivo esperado?
   |
   +-- NO --> Ignorar
   |
   +-- SI --> Confirmar eliminación
```
---

# CONFIRMACIÓN DE ELIMINACIONES

Una eliminación únicamente se registra como **Contenedor Eliminado** cuando eBPF confirma la señal correspondiente al PID esperado.

Flujo:

```text
Daemon
  |
  v
docker stop
  |
  v
Kernel
  |
  v
SIGTERM / SIGKILL
  |
  v
eBPF
  |
  v
Ring Buffer
  |
  v
Daemon
  |
  v
Correlación PID
  |
  v
Valkey
```

Este mecanismo evita registrar como confirmada una eliminación únicamente porque `docker stop` fue solicitado.

---

# GRAFANA

Grafana consume la información almacenada en Valkey mediante el datasource configurado.

El dashboard se denomina:

```text
PANEL DE CONTENEDORES POR 202112092
```

## Paneles implementados

- Total RAM.
- RAM usada.
- Memoria libre.
- Uso de RAM a lo largo del tiempo.
- Total de contenedores eliminados.
- Contenedores eliminados a lo largo del tiempo.
- Top 5 histórico por RAM.
- Top 5 histórico por CPU.
- Contador de eventos eBPF.

## Dashboard exportado

El JSON se encuentra en:

```text
proyecto2/monitoring/grafana/dashboards/dashboard-202112092.json
```
---

# PRUEBA END-TO-END

La prueba final valida todo el flujo del proyecto.

```text
Cronjob
   |
   v
Contenedores
   |
   v
Kernel /proc
   |
   v
Daemon Go
   |
   v
Gestión automática
   |
   v
eBPF
   |
   v
Valkey
   |
   v
Grafana
```

## 21.1 Restricciones finales

```text
LOW >= 3
HIGH >= 2
```

---

# PROCEDIMIENTO DE EJECUCIÓN COMPLETO

## Paso 1 — Levantar monitoring

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/monitoring

docker compose up -d
```

## Paso 2 — Compilar/cargar Kernel

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/kernel

make
./load_module.sh
```

## Paso 3 — Compilar eBPF

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/ebpf

./build.sh
```

## Paso 4 — Compilar Daemon

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/daemon

go mod tidy
go build -o daemon-so1 .
```

## Paso 5 — Ejecutar

```bash
sudo ./daemon-so1
```

## Paso 6 — Abrir Grafana

```text
http://localhost:3000
```

---

# PROCEDIMIENTO DE APAGADO

## Paso 1 — Detener Daemon

Presionar:

```text
Ctrl + C
```

El Daemon debe:

- eliminar el Cronjob;
- cerrar el listener eBPF;
- liberar programas eBPF;
- finalizar de forma controlada.

## Paso 2 — Limpiar contenedores de carga

```bash
docker ps -aq \
--filter "label=so1.project=202112092" \
| xargs -r docker rm -f
```

## Paso 3 — Detener monitoring

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/monitoring

docker compose down
```

No utilizar:

```bash
docker compose down -v
```

si se desean conservar los volúmenes.

## Paso 4 — Descargar módulo Kernel

```bash
cd ~/202112092_LAB_SO1_1S2026/proyecto2/kernel

./unload_module.sh
```

---

# VERIFICACIONES DE LIMPIEZA

## Daemon

```bash
pgrep -af daemon-so1
```

## Cronjob

```bash
sudo crontab -l | grep SO1_P2_202112092
```

## eBPF

```bash
sudo bpftool prog show \
| grep -E "trace_kill|trace_signal"
```

## Kernel

```bash
lsmod | grep continfo
```

## Docker

```bash
docker ps
```

---

# SOLUCIÓN DE PROBLEMAS

## `Key was rejected by service`

Causa probable:

```text
Secure Boot habilitado
```

Verificar:

```bash
mokutil --sb-state
```

Solución: deshabilitar Secure Boot desde BIOS/UEFI o utilizar módulos firmados.

---

## `bpf/bpf_helpers.h file not found`

Instalar:

```bash
sudo apt install -y libbpf-dev
```

Verificar:

```bash
ls -l /usr/include/bpf/bpf_helpers.h
```

---

## El Daemon no conecta con Valkey

Verificar:

```bash
docker exec valkey-so1-202112092 valkey-cli ping
```

Debe responder:

```text
PONG
```

---

## Grafana no encuentra datos

Verificar las claves:

```bash
docker exec valkey-so1-202112092 \
valkey-cli --scan \
--pattern 'so1:202112092:*'
```

---

## No se recibe confirmación eBPF

Verificar que los programas estén cargados:

```bash
sudo bpftool prog show \
| grep -E "trace_kill|trace_signal"
```

Verificar también que el objeto `kill_monitor.bpf.o` haya sido recompilado con `build.sh`.

---

## `Data is missing a number field` en Grafana

Los valores provenientes de streams pueden ser interpretados como texto.

Aplicar la transformación:

```text
Convert field type
→ Numeric
```

sobre el campo que será utilizado en la gráfica.

---

# MANTENIMIENTO

Los artefactos generados no deben almacenarse directamente en Git.

El `.gitignore` excluye elementos como:

```text
daemon-so1
ebpf-test
*.bpf.o
vmlinux.h
*.ko
*.o
*.cmd
*.mod
Module.symvers
modules.order
cronjob.log
```

Esto permite mantener únicamente el código fuente y scripts necesarios para reconstruir el proyecto.

---

# CONCLUSIONES

1. Se implementó correctamente un sistema que integra funcionalidades de espacio de Kernel y espacio de usuario para monitorear procesos y contenedores Docker.

2. El uso de un módulo Kernel permitió obtener métricas de memoria y procesos directamente desde Linux y exponerlas mediante `/proc` para ser consumidas por el Daemon.

3. La integración de eBPF permitió validar las señales de terminación a nivel del Kernel, evitando registrar eliminaciones únicamente por una solicitud de Docker.

4. El Daemon desarrollado en Go centralizó la administración del sistema, relacionando procesos con contenedores, aplicando restricciones y almacenando telemetría en Valkey.

5. Valkey permitió conservar información actual e histórica, mientras que Grafana proporcionó una visualización clara del comportamiento del sistema y de los eventos registrados.

6. Las pruebas finales demostraron que el sistema mantiene las restricciones mínimas de contenedores y puede ejecutar el flujo completo de generación, monitoreo, análisis, eliminación, confirmación y visualización.

---

