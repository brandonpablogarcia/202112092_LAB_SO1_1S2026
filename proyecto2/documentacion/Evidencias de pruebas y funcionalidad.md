UNIVESIDAD DE SAN CARLOS DE GUATEMALA

FACULTAD DE INGENIERIA

ESCUELA DE CIENCAS Y SISTEMAS

LABORATORIO SISTEMAS OPERATIVOS 1 

SECCIÓN P

SEGUNDO SEMESTRE 2026

AUX. JOSÉ DANIEL LORENZANA MEDINA




<p align="center"> EVIDENCIAS DE PRUEBA Y FUNCIONALIDAD </p>



BRANDON EDUARDO PABLO GARCIA

202112092

Guatemala

---

# Introducción

El presente documento reúne las evidencias de prueba y la evidencia funcional del Proyecto 2
Las pruebas fueron realizadas sobre el sistema completo, verificando la integración entre:

- Contenedores Docker.
- Cronjob.
- Módulo Kernel desarrollado en C.
- Archivo virtual en `/proc`.
- Daemon desarrollado en Go.
- Gestión automática de contenedores.
- eBPF.
- Valkey.
- Grafana.

El objetivo de las evidencias es demostrar que cada componente funciona de forma individual y que, al integrarlos, el sistema cumple correctamente con el flujo esperado.

---

# Evidencias de prueba

## Prueba del módulo Kernel

### Objetivo

- Verificar que el módulo Kernel compile, pueda cargarse correctamente y exponga la información requerida mediante `/proc`.

### Resultado esperado

El módulo debe aparecer cargado en el sistema y debe existir:

```text
/proc/continfo_pr2_so1_202112092
```

El contenido debe incluir información de memoria y procesos.

### Evidencia 1 — Procesos obtenidos desde Kernel

[![P2-SO1-202112092-22-Kernel-Procesos-General.png](https://i.postimg.cc/y817LSRC/P2-SO1-202112092-22-Kernel-Procesos-General.png)](https://postimg.cc/PP7GxJKS)

**La captura debe mostrar:**

- Archivo `/proc` leído correctamente.
- Lista de procesos.
- PID.
- Nombre.
- VSZ.
- RSS.
- Porcentaje de RAM.
- Porcentaje de CPU.

---

### Evidencia 2 — Detección de proceso asociado a contenedor

[![P2-SO1-202112092-23-Contenedor-Detectado-Kernel.png](https://i.postimg.cc/pdgbM9mK/P2-SO1-202112092-23-Contenedor-Detectado-Kernel.png)](https://postimg.cc/06dHpydN)


- Un contenedor Docker activo.
- Su PID.
- El mismo PID localizado dentro de la información generada por el módulo Kernel.

---

### Evidencia 3 — Porcentaje de CPU obtenido por Kernel

[![P2-SO1-202112092-24-CPUDesde-Kernel.png](https://i.postimg.cc/5yss1Gsz/P2-SO1-202112092-24-CPUDesde-Kernel.png)](https://postimg.cc/YLm6zXK2)

- Lecturas consecutivas del archivo `/proc`.
- Valor de CPU calculado para uno o más procesos.

---

## Pruebas del Daemon en Go

### Objetivo

Verificar que el Daemon pueda leer la información generada por el Kernel, interpretar los datos y ejecutar ciclos periódicos de monitoreo.


### Evidencia 4 — Lectura y deserialización de `/proc`

[![P2-SO1-202112092-25-Daemon-Parseo-Proc.png](https://i.postimg.cc/4xDBJpQj/P2-SO1-202112092-25-Daemon-Parseo-Proc.png)](https://postimg.cc/TKcr0LSc)

- Inicio del Daemon.
- Lectura del archivo `/proc`.
- Datos interpretados correctamente.
---

### Evidencia 5 — Ejecución periódica del Daemon

[![P2-SO1-202112092-26-Daemon-Loop30Segundos.png](https://i.postimg.cc/L58VyDm1/P2-SO1-202112092-26-Daemon-Loop30Segundos.png)](https://postimg.cc/k6L8Gx6J)

- Al menos dos ciclos consecutivos.
- Intervalo aproximado de 30 segundos.
- Lectura de información en cada ciclo.


---

### Evidencia 6 — Terminación controlada

[![P2-SO1-202112092-27-Daemon-Finalizacion-SIGINT.png](https://i.postimg.cc/DzQdnHZP/P2-SO1-202112092-27-Daemon-Finalizacion-SIGINT.png)](https://postimg.cc/McTjYs1n)

- Uso de `Ctrl + C`.
- Captura de la señal.
- Finalización limpia del Daemon.

---

## Inicialización automática

### Objetivo

Verificar que el Daemon sea capaz de preparar automáticamente los servicios necesarios al iniciar.

### Evidencia 7 — Inicialización de servicios

[![P2-SO1-202112092-28-Daemon-Inicializacion-Automatica.png](https://i.postimg.cc/CK0kP743/P2-SO1-202112092-28-Daemon-Inicializacion-Automatica.png)](https://postimg.cc/9DxDzGKb)

- Inicio de Valkey/Grafana.
- Carga o verificación del módulo Kernel.
- Instalación del Cronjob.
- Inicialización del componente eBPF.

---

## Pruebas del Cronjob

### Objetivo

Comprobar que el Cronjob genere contenedores automáticamente cada dos minutos.

### Evidencia 8 — Generación automática

[![P2-SO1-202112092-29-Daemon-Cron-Contenedores.png](https://i.postimg.cc/QVLHBNYG/P2-SO1-202112092-29-Daemon-Cron-Contenedores.png)](https://postimg.cc/rRJ83TLQ)

- Cronjob instalado.
- Registro del script ejecutado.
- Nuevos contenedores creados.

---

### Evidencia 9 — Eliminación del Cronjob

[![P2-SO1-202112092-30-Daemon-Elimina-Cronjob.png](https://i.postimg.cc/2S77fg3Y/P2-SO1-202112092-30-Daemon-Elimina-Cronjob.png)](https://postimg.cc/nXMD4RvS)

- Finalización del Daemon.
- Verificación de que el Cronjob ya no existe.

---

## Integración Docker — Kernel

### Objetivo

Comprobar que el Daemon pueda asociar correctamente los contenedores Docker con la información de procesos entregada por el Kernel.

### Evidencia 10 — Relación PID Docker / Kernel

[![P2-SO1-202112092-31-Docker-PIDKernel.png](https://i.postimg.cc/wMND4GSm/P2-SO1-202112092-31-Docker-PIDKernel.png)](https://postimg.cc/p529pqVW)

- Nombre del contenedor.
- PID reportado por Docker.
- PID encontrado en el módulo Kernel.

---

## Clasificación de contenedores

### Objetivo

Validar que los perfiles LOW, HIGH RAM, HIGH CPU e INTRUDER sean identificados correctamente.

---

### Evidencia 12 — Clasificación realizada por el Daemon

[![P2-SO1-202112092-33-Clasificacion-Contenedores.png](https://i.postimg.cc/Xq9ykyCY/P2-SO1-202112092-33-Clasificacion-Contenedores.png)](https://postimg.cc/xNTC085w)

- LOW.
- HIGH RAM.
- HIGH CPU.
- INTRUDER.

---

## Pruebas de consumo y rankings

### Objetivo

Comprobar que el sistema pueda ordenar contenedores de acuerdo con el consumo de recursos.

### Evidencia 13 — Ranking de recursos

[![P2-SO1-202112092-34-Rankings-Recursos.png](https://i.postimg.cc/wTV1xzc8/P2-SO1-202112092-34-Rankings-Recursos.png)](https://postimg.cc/vc4YXpN3)

- RAM.
- VSZ.
- RSS.
- CPU.
- Ordenamiento o puntuación de los contenedores.

---

## Selección de contenedores candidatos

### Objetivo

Verificar que el sistema pueda determinar cuáles contenedores pueden eliminarse sin violar los mínimos establecidos.

### Evidencia 14 — Selección previa de candidatos

[![P2-SO1-202112092-35-Candidatos-Dry-Run.png](https://i.postimg.cc/J0z9bjgP/P2-SO1-202112092-35-Candidatos-Dry-Run.png)](https://postimg.cc/KkCqb3mM)

---

## Gestión automática

### Objetivo

Validar que el sistema cree o elimine contenedores según el estado actual.

### Evidencia 15 — Gestión autónoma

[![P2-SO1-202112092-36-Gestion-Autonoma.png](https://i.postimg.cc/52CDq4PK/P2-SO1-202112092-36-Gestion-Autonoma.png)](https://postimg.cc/sM3n3rC5)

- Estado antes de la gestión.
- Acción ejecutada.
- Estado posterior.

---

## Validación de restricciones

### Objetivo

Comprobar que el sistema conserve como mínimo:

```text
LOW >= 3
HIGH >= 2
```

donde:

```text
HIGH = HIGH_RAM + HIGH_CPU
```

### Evidencia 16 — Restricciones cumplidas

[![P2-SO1-202112092-37-Restricciones-Cumplidas.png](https://i.postimg.cc/Gpj7qHvP/P2-SO1-202112092-37-Restricciones-Cumplidas.png)](https://postimg.cc/PLPQqf3N)
---

# Evidencias de Valkey

## Claves generadas

### Objetivo

Comprobar que el Daemon almacene correctamente la información en Valkey.

### Evidencia 17 — Claves almacenadas

[![P2-SO1-202112092-38-Valkey-Claves-Daemon.png](https://i.postimg.cc/VkQKwjRZ/P2-SO1-202112092-38-Valkey-Claves-Daemon.png)](https://postimg.cc/mtjYN1K7)

---

## Telemetría

### Evidencia 18 — Información almacenada

[![P2-SO1-202112092-39-Telemetria-Valkey.png](https://i.postimg.cc/bvMT6kst/P2-SO1-202112092-39-Telemetria-Valkey.png)](https://postimg.cc/VSqnvS9f)

---

## Historial y rankings

### Evidencia 19 — Información histórica

- Historial de memoria.
- Ranking histórico de RAM.
- Ranking histórico de CPU.

---

# Evidencias eBPF

## Compilación del programa

### Objetivo

Verificar que el programa eBPF compile correctamente.

### Evidencia 20 — Compilación eBPF

[![P2-SO1-202112092-41-e-BPFCompilacion.png](https://i.postimg.cc/prBKtFXg/P2-SO1-202112092-41-e-BPFCompilacion.png)](https://postimg.cc/4HmY6nc5)
---

## Intercepción de señales

### Objetivo

Verificar que eBPF pueda interceptar señales enviadas a los procesos.

### Evidencia 21 — Intercepción de kill

[![P2-SO1-202112092-42-e-BPFKill-Interceptado.png](https://i.postimg.cc/QxXpJ0fF/P2-SO1-202112092-42-e-BPFKill-Interceptado.png)](https://postimg.cc/K4HkZrSb)
---

## Prueba con Docker

### Evidencia 22 — Señal generada durante Docker Stop

- PID objetivo.
- Señal SIGTERM o SIGKILL.
- Proceso emisor.

---

## Integración eBPF con Daemon

### Evidencia 23 — Integración con Go

[![P2-SO1-202112092-44-e-BPFDaemon-Integrado.png](https://i.postimg.cc/GtzTfnqW/P2-SO1-202112092-44-e-BPFDaemon-Integrado.png)](https://postimg.cc/RJ603yrd)
---

## Confirmación real de eliminación

### Objetivo

Demostrar que una eliminación solamente se registra cuando el Kernel confirma la señal mediante eBPF.

### Evidencia 24 — Confirmación eBPF

[![P2-SO1-202112092-45-e-BPFConfirmacion-Valkey.png](https://i.postimg.cc/Ghhyy9pF/P2-SO1-202112092-45-e-BPFConfirmacion-Valkey.png)](https://postimg.cc/xqh8rfM8)

---

# Evidencias de Grafana

## Conexión Grafana — Valkey

### Objetivo

Comprobar que Grafana pueda conectarse correctamente al origen de datos.

---

## Métricas generales de memoria

### Evidencia 27 — RAM

[![P2-SO1-202112092-48-Grafana-RAMGeneral.png](https://i.postimg.cc/xTCXd7sC/P2-SO1-202112092-48-Grafana-RAMGeneral.png)](https://postimg.cc/Cd3hPvpT)

- RAM total.
- RAM usada.
- RAM libre.

---

## RAM a lo largo del tiempo

### Evidencia 28 — Serie temporal

[![P2-SO1-202112092-49-Grafana-Evolucion-RAM.png](https://i.postimg.cc/Vk05qtP8/P2-SO1-202112092-49-Grafana-Evolucion-RAM.png)](https://postimg.cc/3k7K7dwS)
---

## Rankings históricos

### Evidencia 29 — Top 5

[![P2-SO1-202112092-50-Grafana-Top5.png](https://i.postimg.cc/xC5dfyMW/P2-SO1-202112092-50-Grafana-Top5.png)](https://postimg.cc/ygJKnRfP)

- Top 5 de RAM.
- Top 5 de CPU.

---

## Eventos eBPF

### Evidencia 30 — Contador de eventos

[![P2-SO1-202112092-51-Grafana-Eventos-EBPF.png](https://i.postimg.cc/0QV8R51w/P2-SO1-202112092-51-Grafana-Eventos-EBPF.png)](https://postimg.cc/zbRmTrBz)

---

## Dashboard completo

### Evidencia 31 — Panel completo

[![P2-SO1-202112092-52-Dashboard-Completo.png](https://i.postimg.cc/zvM8vdBn/P2-SO1-202112092-52-Dashboard-Completo.png)](https://postimg.cc/JHb9d5jn)

- Total RAM.
- RAM utilizada.
- RAM libre.
- Histórico de RAM.
- Eliminaciones.
- Top 5 RAM.
- Top 5 CPU.
- Eventos eBPF.

---

# Flujo funcional 

Durante la prueba final se comprobó el siguiente flujo:

```text
Cronjob
   ↓
Creación de contenedores
   ↓
Docker
   ↓
Kernel Module
   ↓
/proc/continfo_pr2_so1_202112092
   ↓
Daemon Go
   ↓
Clasificación y análisis
   ↓
Gestión automática
   ↓
docker stop
   ↓
eBPF
   ↓
Confirmación Kernel
   ↓
Valkey
   ↓
Grafana
```

---

## Restricciones finales

### Evidencia 32 — Estado final de contenedores

[![P2-SO1-202112092-53-Restricciones-Finales.png](https://i.postimg.cc/cJmWw1CG/P2-SO1-202112092-53-Restricciones-Finales.png)](https://postimg.cc/DW8RKT1x)

- LOW >= 3
- HIGH_RAM + HIGH_CPU >= 2

También debe permitir comprobar que los servicios de infraestructura, como Grafana y Valkey, continúan activos.

---

## Persistencia de eventos confirmados

### Evidencia 33 — Evento eBPF final en Valkey

[![P2-SO1-202112092-54-Valkey-Final-EBPF.png](https://i.postimg.cc/GmnnfTTf/P2-SO1-202112092-54-Valkey-Final-EBPF.png)](https://postimg.cc/mtjnzDw3)

- Contenedor eliminado.
- PID objetivo.
- Señal recibida.
- Contador de eventos.
- Registro persistido en Valkey.

---

## Dashboard funcional final

### Evidencia 34 — Dashboard 

[![P2-SO1-202112092-55-Dashboard-Final.png](https://i.postimg.cc/nhKffVv2/P2-SO1-202112092-55-Dashboard-Final.png)](https://postimg.cc/Wd4HMvYq)

La captura debe mostrar el dashboard final con información actualizada de:

- RAM total.
- RAM utilizada.
- RAM libre.
- Histórico de RAM.
- Total de eliminaciones.
- Eliminaciones a lo largo del tiempo.
- Top 5 RAM.
- Top 5 CPU.
- Contador de eventos eBPF.

---

## Liberación de recursos

### Objetivo

Comprobar que el sistema finalice correctamente y no deje recursos activos innecesarios.

### Evidencia 35 — Limpieza final

[![P2-SO1-202112092-56-Limpieza-Final.png](https://i.postimg.cc/FzkwWhrF/P2-SO1-202112092-56-Limpieza-Final.png)](https://postimg.cc/gLdMrFL1)

- Daemon finalizado.
- Cronjob eliminado.
- Programas eBPF liberados.
- Terminación limpia.


---

# Resultado de la prueba funcional

La prueba funcional final fue completada satisfactoriamente.

El sistema demostró que puede:

1. Generar contenedores automáticamente.
2. Obtener información de procesos desde el Kernel.
3. Relacionar procesos con contenedores Docker.
4. Clasificar contenedores según su perfil.
5. Analizar consumo de RAM, VSZ, RSS y CPU.
6. Mantener los mínimos requeridos de contenedores.
7. Seleccionar candidatos para eliminación.
8. Ejecutar la eliminación desde el Daemon.
9. Confirmar la señal de terminación mediante eBPF.
10. Registrar únicamente eliminaciones confirmadas.
11. Almacenar información actual e histórica en Valkey.
12. Visualizar la información mediante Grafana.
13. Finalizar correctamente y liberar los recursos asociados.

---

# Conclusión

Las pruebas realizadas permiten comprobar el funcionamiento individual e integrado de los componentes desarrollados para el Proyecto 2. El módulo Kernel proporciona la información requerida sobre memoria y procesos; el Daemon procesa y utiliza dichos datos para administrar los contenedores; eBPF permite validar eventos de terminación directamente desde el Kernel; Valkey mantiene la información actual e histórica; y Grafana permite visualizar el estado general del sistema.

La prueba de integración final confirmó el funcionamiento completo del flujo, desde la creación automática de contenedores hasta la visualización y persistencia de los resultados. Además, se verificó que el sistema conserva los mínimos definidos, protege los servicios de infraestructura y libera correctamente los recursos al finalizar.
Por lo tanto, las evidencias recopiladas permiten demostrar que la implementación cumple funcionalmente con los objetivos establecidos para el proyecto.


