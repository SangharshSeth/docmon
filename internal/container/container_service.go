package container

import (
	"context"

	"github.com/sangharshseth/docmon/internal/types"
)

// ContainerService defines the interface for container operations
type ContainerService interface {
	// ListContainers returns a list of all containers with their details
	GetAllContainers(ctx context.Context) ([]types.DockerContainerBasicInfo, error)
}
