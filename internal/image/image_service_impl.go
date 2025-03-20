package image

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/sangharshseth/docmon/internal/docker"
	"github.com/sangharshseth/docmon/internal/types"
)

type imageServiceImpl struct {
	DockerManager *docker.DockerManager
}

func NewImageServiceImpl(dockerManager *docker.DockerManager) *imageServiceImpl {
	return &imageServiceImpl{
		DockerManager: dockerManager,
	}
}

func (i *imageServiceImpl) ListImages(ctx context.Context) ([]types.DockerImageDetails, error) {
	var image_ids []string
	var image_details []types.DockerImageDetails
	c, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	images, err := i.DockerManager.DockerClient.ImageList(c, image.ListOptions{All: true})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("Image list operation timedout")
		}
		return nil, fmt.Errorf("%s", err.Error())
	}
	for _, image := range images {
		image_ids = append(image_ids, image.ID)
	}

	//Fetch inspect
	for _, image_id := range image_ids {
		c, cancel := context.WithTimeout(ctx, time.Second*2)
		defer cancel()
		inspect, err := i.DockerManager.DockerClient.ImageInspect(c, image_id)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				slog.Error("Image inspect operation timedout")
			}
			return nil, fmt.Errorf("%s", err.Error())
		}
		image_details = append(image_details, types.DockerImageDetails{
			ID:        strings.TrimPrefix(inspect.ID, "sha256:")[:12],
			RepoTags:  inspect.RepoTags,
			CreatedAt: inspect.Created,
			Size:      fmt.Sprintf("%.2f MB", float64(inspect.Size)/float64(1024*1024)),
			Arch:      inspect.Architecture,
			OS:        inspect.Os,
			Labels:    inspect.Config.Labels,
		})
	}

	return image_details, nil
}
