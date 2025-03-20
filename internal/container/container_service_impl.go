package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sangharshseth/docmon/internal/docker"
	"github.com/sangharshseth/docmon/internal/types"
)

type ContainerServiceImpl struct {
	DockerManager *docker.DockerManager
}

func NewContainerServiceImpl(dockerManager *docker.DockerManager) *ContainerServiceImpl {
	return &ContainerServiceImpl{
		DockerManager: dockerManager,
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
	var docker_container_basic_info []types.DockerContainerBasicInfo
	ctxC, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	container, err := csrv.DockerManager.DockerClient.ContainerList(ctxC, container.ListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("Container list operation timedout")
		}
		return nil, fmt.Errorf("%s", err.Error())
	}
	for _, c := range container {
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
	return docker_container_basic_info, nil

}
