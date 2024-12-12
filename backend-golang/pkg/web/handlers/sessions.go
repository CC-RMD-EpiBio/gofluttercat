package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	rdb     *redis.Client
	models  map[string]irt.GradedResponseModel
	Context context.Context
}

type Answer struct {
	Name     string
	Response int
}

func NewHandler(rdb *redis.Client, models map[string]irt.GradedResponseModel) SessionHandler {
	return SessionHandler{
		rdb:    rdb,
		models: models,
	}
}

func (sh *SessionHandler) GetCatSession(writer http.ResponseWriter, request *http.Request) {
	id := uuid.New()
	sess := &models.SessionState{
		SessionId:  id.String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
	}

	prior := func(x float64) float64 {
		m := math2.NewGaussianDistribution(0, 10)
		return m.Density(x)
	}

	// initialize the CAT session
	scorers := make(map[string]*models.BayesianScorer, 0)
	for label, m := range sh.models {
		scorers[label] = models.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), prior, m)
	}

	// write record of this session
	err := respondWithJSON(writer, http.StatusOK, sess)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

}

func (sh *SessionHandler) DeactivateCatSession(writer http.ResponseWriter, request *http.Request) {

}

func (sh *SessionHandler) NewCatSession(writer http.ResponseWriter, request *http.Request) {

	writer.WriteHeader(http.StatusOK)

}
