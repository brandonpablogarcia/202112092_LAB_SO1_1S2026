package models

/*
MemoryInfo representa la información general
de memoria RAM obtenida desde el módulo Kernel.
*/
type MemoryInfo struct {
	TotalKB uint64 `json:"total_kb"`
	FreeKB  uint64 `json:"free_kb"`
	UsedKB  uint64 `json:"used_kb"`
}

/*
ProcessInfo representa cada proceso detectado
mediante task_struct en el módulo Kernel.
*/
type ProcessInfo struct {
	PID           int     `json:"pid"`
	Name          string  `json:"name"`
	CommandLine   string  `json:"command_line"`
	VSZKB         uint64  `json:"vsz_kb"`
	RSSKB         uint64  `json:"rss_kb"`
	MemoryPercent float64 `json:"memory_percent"`
	CPUPercent    float64 `json:"cpu_percent"`
}

/*
Telemetry representa la estructura completa
expuesta por /proc/continfo_pr2_so1_202112092.
*/
type Telemetry struct {
	Carnet       string        `json:"carnet"`
	Memory       MemoryInfo    `json:"memory"`
	Processes    []ProcessInfo `json:"processes"`
	ProcessCount int           `json:"process_count"`
}
