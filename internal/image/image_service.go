package image

import (
	"context"

	"github.com/sangharshseth/docmon/internal/types"
)

type ImageService interface {
	getAllImages(c context.Context) ([]types.DockerImageDetails, error)
}
