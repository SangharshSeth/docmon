package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/redis/go-redis/v9"
	"github.com/sangharshseth/docmon/internal/docker"
	"github.com/sangharshseth/docmon/internal/types"
)

type ContainerServiceImpl struct {
	DockerManager *docker.DockerManager
	RedisClient   *redis.Client
}

type ContainerResponse struct {
	Snapshots []types.DockerContainerBasicInfo `json:"snapshots"`
	Inspect   []types.DockerContainerInspect   `json:"inspect"`
}

func NewContainerServiceImpl(dockerManager *docker.DockerManager, redisClient *redis.Client) *ContainerServiceImpl {
	return &ContainerServiceImpl{
		DockerManager: dockerManager,
		RedisClient:   redisClient,
	}
}

func parsePortMappings(ports []container.Port) []types.PortMapping {
	var portMappings []types.PortMapping
	for _, port := range ports {
		if port.IP != "" {
			portMappings = append(portMappings, types.PortMapping{
				IP:          port.IP,
				PrivatePort: int(port.PrivatePort),
				PublicPort:  int(port.PublicPort),
				Type:        port.Type,
			})
		}
	}
	return portMappings
}

func (csrv *ContainerServiceImpl) GetAllContainers(ctx context.Context) ([]types.DockerContainerBasicInfo, error) {
	// Try to get from cache first
	rctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	// Check cache first
	cachedData, err := csrv.RedisClient.Get(rctx, "container-list").Bytes()
	if err == nil {
		var cachedContainers []types.DockerContainerBasicInfo
		if err := json.Unmarshal(cachedData, &cachedContainers); err == nil {
			slog.Info("Cache hit: Returning containers from Redis")
			return cachedContainers, nil
		}
		slog.Error("Failed to unmarshal cached container data", "error", err)
	}

	slog.Info("Cache miss: Fetching containers from Docker")

	var docker_container_basic_info []types.DockerContainerBasicInfo

	ctxC, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	containers, err := csrv.DockerManager.DockerClient.ContainerList(ctxC, container.ListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("Container list operation timedout")
		}
		return nil, fmt.Errorf("%s", err.Error())
	}

	for _, c := range containers {
		basic_info := types.DockerContainerBasicInfo{
			ID:        c.ID[:12],
			Name:      c.Names,
			Image:     c.Image,
			Command:   c.Command,
			CreatedAt: time.Since(time.Unix(int64(c.Created), 0)).Round(time.Second),
			Status:    c.Status,
			State:     c.State,
			Ports:     parsePortMappings(c.Ports),
		}
		docker_container_basic_info = append(docker_container_basic_info, basic_info)
	}

	// Store in Redis with 5 minute TTL
	jsonData, err := json.Marshal(docker_container_basic_info)
	if err != nil {
		slog.Error("Failed to marshal container details", "error", err)
		return docker_container_basic_info, nil
	}

	rctx, cancel = context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	err = csrv.RedisClient.Set(rctx, "container-list", jsonData, 5*time.Minute).Err()
	if err != nil {
		slog.Error("Failed to store container data in Redis", "error", err)
	} else {
		slog.Info("Successfully stored container data in Redis", "key", "container-list", "size", len(jsonData))
	}

	return docker_container_basic_info, nil
}

func (csrv *ContainerServiceImpl) GetContainerInspectById(ctx context.Context, id string) (types.DockerContainerInspect, error) {
	inspect, err := csrv.DockerManager.DockerClient.ContainerInspect(ctx, id)
	if err != nil {
		return types.DockerContainerInspect{}, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Convert to our simplified structure
	networks := make([]types.NetworkInfo, 0)
	for name, network := range inspect.NetworkSettings.Networks {
		networks = append(networks, types.NetworkInfo{
			NetworkName: name,
			IPAddress:   network.IPAddress,
			Gateway:     network.Gateway,
		})
	}

	return types.DockerContainerInspect{
		ID:           inspect.ID[:12],
		Name:         inspect.Name,
		Created:      inspect.Created,
		Platform:     inspect.Platform,
		RestartCount: inspect.RestartCount,
		State: types.SimpleState{
			Status:     inspect.State.Status,
			Running:    inspect.State.Running,
			StartedAt:  inspect.State.StartedAt,
			FinishedAt: inspect.State.FinishedAt,
			ExitCode:   inspect.State.ExitCode,
		},
		Networks: networks,
	}, nil
}

func (csrv *ContainerServiceImpl) GetContainerStatsById(ctx context.Context, id string) (*types.SystemStats, error) {
	// Get container stats from Docker
	stats, err := csrv.DockerManager.DockerClient.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var statsData types.SystemStats
	var dockerStats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
		Networks struct {
			Eth0 struct {
				RxBytes uint64 `json:"rx_bytes"`
				TxBytes uint64 `json:"tx_bytes"`
			} `json:"eth0"`
		} `json:"networks"`
	}

	if err := json.NewDecoder(stats.Body).Decode(&dockerStats); err != nil {
		return nil, fmt.Errorf("failed to decode stats JSON: %w", err)
	}

	// Calculate CPU usage percentage
	cpuDelta := float64(dockerStats.CPUStats.CPUUsage.TotalUsage - dockerStats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(dockerStats.CPUStats.SystemCPUUsage - dockerStats.PreCPUStats.SystemCPUUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		statsData.CPUUsage = (cpuDelta / systemDelta) * 100
	}

	// Calculate memory usage percentage
	if dockerStats.MemoryStats.Limit > 0 {
		statsData.MemoryUsage = (float64(dockerStats.MemoryStats.Usage) / float64(dockerStats.MemoryStats.Limit)) * 100
	}

	// Network stats
	statsData.RXBytes = float64(dockerStats.Networks.Eth0.RxBytes)
	statsData.TXBytes = float64(dockerStats.Networks.Eth0.TxBytes)

	return &statsData, nil
}

func (csrv *ContainerServiceImpl) GetTotalResourceUsageByAllContainers(ctx context.Context) (*types.AggregatedSystemStats, error) {
	// Try to get from the cache first

	// Get list of all containers
	containers, err := csrv.GetAllContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container list: %w", err)
	}

	aggregatedStats := &types.AggregatedSystemStats{
		PerContainer: make(map[string]*types.SystemStats),
	}

	// Collect stats for each container
	for _, container := range containers {
		// Skip if the container is not running
		if container.State != "running" {
			continue
		}

		stats, err := csrv.GetContainerStatsById(ctx, container.ID)
		if err != nil {
			slog.Error("Failed to get stats for container",
				"containerId", container.ID,
				"error", err)
			continue
		}

		// Add to per-container map
		aggregatedStats.PerContainer[container.ID] = stats

		// Aggregate statistics
		aggregatedStats.TotalCPUUsage += stats.CPUUsage
		aggregatedStats.TotalMemoryUsage += stats.MemoryUsage
		aggregatedStats.TotalRXBytes += stats.RXBytes
		aggregatedStats.TotalTXBytes += stats.TXBytes
		aggregatedStats.RunningCount++
	}

	aggregatedStats.ContainerCount = len(containers)

	// Cache the results
	_, err = json.Marshal(aggregatedStats)
	if err != nil {
		slog.Error("Failed to marshal aggregated stats", "error", err)
		return aggregatedStats, nil
	}
	return aggregatedStats, nil
}
