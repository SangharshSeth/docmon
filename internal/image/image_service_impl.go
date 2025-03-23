package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/redis/go-redis/v9"
	"github.com/sangharshseth/docmon/internal/docker"
	"github.com/sangharshseth/docmon/internal/types"
)

type imageServiceImpl struct {
	DockerManager *docker.DockerManager
	RedisClient   *redis.Client
}

func NewImageServiceImpl(dockerManager *docker.DockerManager, redisClient *redis.Client) *imageServiceImpl {
	return &imageServiceImpl{
		DockerManager: dockerManager,
		RedisClient:   redisClient,
	}
}

func (i *imageServiceImpl) ListImages(ctx context.Context) ([]types.DockerImageDetails, error) {
	// Try to get from cache first
	rctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	// Check cache first
	cachedData, err := i.RedisClient.Get(rctx, "image-list").Bytes()
	if err == nil {
		var cachedImages []types.DockerImageDetails
		if err := json.Unmarshal(cachedData, &cachedImages); err == nil {
			slog.Info("Cache hit: Returning images from Redis")
			return cachedImages, nil
		}
		slog.Error("Failed to unmarshal cached data", "error", err)
	}

	slog.Info("Cache miss: Fetching images from Docker")

	// Rest of the existing Docker fetching logic
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

	// Convert to JSON for storage
	jsonData, err := json.Marshal(image_details)
	if err != nil {
		slog.Error("Failed to marshal image details", "error", err)
		return image_details, nil
	}

	rctx, cancel = context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	err = i.RedisClient.Set(rctx, "image-list", jsonData, 5*time.Minute).Err()
	if err != nil {
		slog.Error("Failed to store data in Redis", "error", err)
	} else {
		slog.Info("Successfully stored data in Redis", "key", "image-list", "size", len(jsonData))
	}

	return image_details, nil
}
