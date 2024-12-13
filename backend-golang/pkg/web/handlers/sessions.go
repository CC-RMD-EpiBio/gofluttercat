package handlers

import (
	"context"
	"fmt"
	"log"
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
	models  *map[string]irt.GradedResponseModel
	Context *context.Context
}

type Answer struct {
	Name     string
	Response int
}

func NewSessionHandler(rdb *redis.Client, models map[string]irt.GradedResponseModel, ctx context.Context) SessionHandler {
	return SessionHandler{
		rdb:     rdb,
		models:  &models,
		Context: &ctx,
	}
}

func (sh *SessionHandler) NewCatSession(writer http.ResponseWriter, request *http.Request) {
	id := uuid.New()

	prior := func(x float64) float64 {
		m := math2.NewGaussianDistribution(0, 10)
		return m.Density(x)
	}

	// initialize the CAT session
	scorers := make(map[string]*models.BayesianScorer, 0)
	for label, m := range *sh.models {
		scorers[label] = models.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), prior, m)
	}

	energies := make(map[string][]float64, 0)
	for label, s := range scorers {
		energies[label] = s.Running.Energy
	}

	sess := &models.SessionState{
		SessionId:  id.String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:   energies,
	}

	sbyte, _ := sess.ByteMarshal()

	if sh.Context == nil {
		ctx := context.Background()
		sh.Context = &ctx
	}
	stus := sh.rdb.Set(*sh.Context, sess.SessionId, sbyte, sess.Expiration.Sub(time.Now()))
	err := stus.Err()
	if err != nil {
		log.Printf("err: %v\n", err)
		panic(err)
	}
	log.Printf("New Session: %v\n", sess.SessionId)

	// write record of this session
	out := map[string]string{
		"session_id":      sess.SessionId,
		"start_time":      sess.Start.String(),
		"expiration_time": sess.Expiration.String(),
	}
	err = respondWithJSON(writer, http.StatusOK, out)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

}

func (sh *SessionHandler) DeactivateCatSession(writer http.ResponseWriter, request *http.Request) {

}
