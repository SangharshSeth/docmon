package types

import "time"

type DockerImageDetails struct {
	ID        string            `json:"id"`
	RepoTags  []string          `json:"repo_tags"`
	CreatedAt string            `json:"created_at"`
	Size      string            `json:"size"`
	Arch      string            `json:"arch"`
	OS        string            `json:"os"`
	Labels    map[string]string `json:"labels"`
}

type DockerContainerBasicInfo struct {
	ID        string        // Container ID (shortened to 12 characters)
	Name      []string      // Container name (without the leading "/")
	Image     string        // Image name (e.g., "ubuntu:latest")
	Command   string        // Command executed in the container
	CreatedAt time.Duration // Timestamp when the container was created
	Status    string        // Current status (e.g., "Up 2 hours", "Exited (0) 5 minutes ago")
	State     string        // Current state (e.g., "running", "exited", "paused")
	Ports     []PortMapping // Port mappings (e.g., "0.0.0.0:8080->80/tcp")
}

type PortMapping struct {
	IP          string // Host IP (e.g., "0.0.0.0")
	PrivatePort int    // Container port (e.g., 80)
	PublicPort  int    // Host port (e.g., 8080)
	Type        string // Port type (e.g., "tcp", "udp")
}

type DockerContainerInspect struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Created      string        `json:"created"`
	State        SimpleState   `json:"state"`
	Platform     string        `json:"platform"`
	RestartCount int           `json:"restart_count"`
	Networks     []NetworkInfo `json:"networks"`
}

type SimpleState struct {
	Status     string `json:"status"`      // human readable status (running, stopped, etc)
	Running    bool   `json:"running"`     // whether container is currently running
	StartedAt  string `json:"started_at"`  // when the container was started
	FinishedAt string `json:"finished_at"` // when the container stopped (if not running)
	ExitCode   int    `json:"exit_code"`   // 0 means successful exit, non-zero means error
}

type NetworkInfo struct {
	NetworkName string `json:"network_name"`
	IPAddress   string `json:"ip_address"`
	Gateway     string `json:"gateway"`
}

type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	RXBytes     float64 `json:"rx_bytes"`
	TXBytes     float64 `json:"tx_bytes"`
}

type AggregatedSystemStats struct {
	TotalCPUUsage    float64                 `json:"total_cpu_usage"`
	TotalMemoryUsage float64                 `json:"total_memory_usage"`
	TotalRXBytes     float64                 `json:"total_rx_bytes"`
	TotalTXBytes     float64                 `json:"total_tx_bytes"`
	ContainerCount   int                     `json:"container_count"`
	RunningCount     int                     `json:"running_count"`
	PerContainer     map[string]*SystemStats `json:"per_container"`
}
