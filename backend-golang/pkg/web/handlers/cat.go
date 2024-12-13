package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type CatHandlerHelper struct {
	rdb     *redis.Client
	models  *map[string]irt.GradedResponseModel
	Context *context.Context
}

func NewCatHandlerHelper(rdb *redis.Client, models *map[string]irt.GradedResponseModel, context *context.Context) CatHandlerHelper {
	return CatHandlerHelper{
		rdb:     rdb,
		models:  models,
		Context: context,
	}
}

func (ch *CatHandlerHelper) NextItem(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
}

func (ch *CatHandlerHelper) NextScaleItem(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
	scale := chi.URLParam(request, "scale")

}

func (ch *CatHandlerHelper) RegisterResponse(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
	
}
