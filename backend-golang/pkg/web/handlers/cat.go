package handlers

import (
	"context"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/redis/go-redis/v9"
)

type CatHandlerHelper struct {
	rdb     *redis.Client
	models  *map[string]irt.GradedResponseModel
	Context *context.Context
}
