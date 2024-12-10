package handlers

import (
	"net/http"
)

type SessionHandler struct {
	Id string
}

func (uh *SessionHandler) GetSession(writer http.ResponseWriter, request *http.Request) {

	writer.WriteHeader(http.StatusOK)

}

func (uh *SessionHandler) CreateSession(writer http.ResponseWriter, request *http.Request) {

}

func (uh *SessionHandler) DeactivateSession(writer http.ResponseWriter, request *http.Request) {

}
