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

func (csrv *ContainerServiceImpl) GetContainerInspectById(ctx context.Context, id string) types.DockerContainerInspect {
	inspect, err := csrv.DockerManager.DockerClient.ContainerInspect(ctx, id)
	if err != nil {
		panic(err)
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
	}
}
