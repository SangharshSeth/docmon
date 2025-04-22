package container

import (
	"context"

	"github.com/sangharshseth/docmon/internal/types"
)

// ContainerService defines the interface for container operations
// ContainerService defines the interface for container operations
type ContainerService interface {
	// GetAllContainers returns a list of all containers with their details
	GetAllContainers(ctx context.Context) ([]types.DockerContainerBasicInfo, error)
	// GetContainerStatsById returns resource usage statistics for a specific container
	GetContainerStatsById(ctx context.Context, id string) (*types.SystemStats, error)
	// GetTotalResourceUsageByAllContainers returns aggregated resource usage statistics for all containers
	GetTotalResourceUsageByAllContainers(ctx context.Context) (*types.AggregatedSystemStats, error)
}
