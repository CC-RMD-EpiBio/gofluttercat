package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	rdb    *redis.Client
	models map[string]irt.GradedResponseModel
}

type Session struct {
	Id         string    `json:"session_id"`
	Start      time.Time `json:"start_time"`
	Expiration time.Time `json:"end_time"`
}

func NewHandler(rdb *redis.Client, models map[string]irt.GradedResponseModel) SessionHandler {
	return SessionHandler{
		rdb:    rdb,
		models: models,
	}
}

func (sh *SessionHandler) GetCatSession(writer http.ResponseWriter, request *http.Request) {
	id := uuid.New()
	sess := &Session{
		Id:         id.String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
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
