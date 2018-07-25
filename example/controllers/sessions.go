package controllers

import (
	"net/http"

	"github.com/dimfeld/httptreemux"
)

type sessionsImpl struct{}

type sessionRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func registerSessions(router *httptreemux.TreeMux) {
	impl := &sessionsImpl{}

	router.POST("/sessions", impl.create)
}

func (impl *sessionsImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
}
