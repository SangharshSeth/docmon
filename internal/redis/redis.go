package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func NewClient(opt *redis.Options) *redis.Client {
	return redis.NewClient(opt)
}
