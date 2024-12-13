package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/go-chi/chi/v5"
	"github.com/mederrata/ndvek"
	"github.com/redis/go-redis/v9"
)

type SummaryHandler struct {
	rdb     *redis.Client
	models  *map[string]irt.GradedResponseModel
	Context *context.Context
}

type SessionSummary struct {
	SessionId      string                   `json:"session_id"`
	StartTime      time.Time                `json:"start_time"`
	ExpirationTime time.Time                `json:"expiration_time"`
	Responses      []*models.SkinnyResponse `json:"responses"`
}

type ScoreSummary struct {
	Mean    float64   `json:"mean"`
	Std     float64   `json:"std"`
	Deciles []float64 `json:"deciles"`
	// Density []float64 `json:"density"`
	// Grid    []float64 `json:"grid"`
}
type Summary struct {
	Session SessionSummary          `json:"session"`
	Scores  map[string]ScoreSummary `json:"scores"`
}

func NewSesssionSummary(s models.SessionState) SessionSummary {
	out := SessionSummary{
		SessionId:      s.SessionId,
		StartTime:      s.Start,
		ExpirationTime: s.Expiration,
		Responses:      s.Responses,
	}
	return out
}

func NewScoreSummary(bs *models.BayesianScore) ScoreSummary {
	out := ScoreSummary{
		Mean: bs.Mean(),
		Std:  bs.Std(),
		// Density: bs.Density(),
		// Grid:    bs.Grid,
		Deciles: bs.Deciles(),
	}
	return out
}

func NewSummaryHandler(rdb *redis.Client, models map[string]irt.GradedResponseModel, ctx context.Context) *SummaryHandler {
	return &SummaryHandler{
		rdb:     rdb,
		models:  &models,
		Context: &ctx,
	}
}

func (sh SummaryHandler) ProvideSummary(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, _ := models.SessionStateFromId(sid, *sh.rdb, sh.Context)

	scores := make(map[string]*models.BayesianScore, 0)
	summary := Summary{
		Session: NewSesssionSummary(*rehydrated),
		Scores:  make(map[string]ScoreSummary),
	}
	for label, energy := range rehydrated.Energies {
		scores[label] = &models.BayesianScore{
			Energy: energy,
			Grid:   ndvek.Linspace(-10, 10, 400),
		}
		summary.Scores[label] = NewScoreSummary(scores[label])
	}

	respondWithJSON(writer, http.StatusOK, summary)

}
