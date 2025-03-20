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
