package image

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/sangharshseth/docmon/internal/docker"
)

type imageServiceImpl struct {
	DockerManager *docker.DockerManager
}

func NewImageServiceImpl(dockerManager *docker.DockerManager) *imageServiceImpl {
	return &imageServiceImpl{
		DockerManager: dockerManager,
	}
}

func (i *imageServiceImpl) ListImages(ctx context.Context) ([]image.Summary, error) {
	c, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	images, err := i.DockerManager.DockerClient.ImageList(c, image.ListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("Image list operation timedout")
		}
		return nil, fmt.Errorf("%s", err.Error())
	}
	return images, nil
}
