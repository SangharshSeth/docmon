package docker

import (
	"log/slog"

	"github.com/docker/docker/client"
)

type DockerManager struct {
	DockerClient *client.Client
}

// Create a new DockerManager instance.
func NewDockerManager() (*DockerManager, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("Failed to create Docker client", err)
		return nil, err
	}
	return &DockerManager{
		DockerClient: dockerClient,
	}, nil
}
