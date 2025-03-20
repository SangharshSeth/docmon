package image

import (
	"context"

	"github.com/docker/docker/api/types/image"
)

type ImageService interface {
	getAllImages(c context.Context) ([]image.Summary, error)
}
