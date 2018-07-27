package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
)

type sessionsImpl struct{}

type sessionRequest struct {
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Password      string `json:"password"`
	SessionSecret string `json:"session_secret"`
}

func registerSessions(router *httptreemux.TreeMux) {
	impl := &sessionsImpl{}

	router.POST("/sessions", impl.create)
}

func (impl *sessionsImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	receiver := body.Phone
	if receiver == "" {
		receiver = body.Email
	}
	if receiver == "" {
		views.RenderErrorResponse(w, r, session.BadDataError(r.Context()))
		return
	}

	user, err := models.CreateSession(r.Context(), receiver, body.Password, body.SessionSecret)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}
	views.RenderUserWithAuthentication(w, r, user)
}
