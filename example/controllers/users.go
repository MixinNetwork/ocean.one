package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/middlewares"
	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
)

type usersImpl struct{}

type userRequest struct {
	VerificationId string `json:"verification_id"`
	Password       string `json:"password"`
	SessionSecret  string `json:"session_secret"`
}

func registerUsers(router *httptreemux.TreeMux) {
	impl := &usersImpl{}

	router.POST("/users", impl.create)
	router.GET("/me", impl.me)
}

func (impl *usersImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body userRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	user, err := models.CreateUser(r.Context(), body.VerificationId, body.Password, body.SessionSecret)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}
	views.RenderUserWithAuthentication(w, r, user)
}

func (impl *usersImpl) me(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	views.RenderUserWithAuthentication(w, r, middlewares.CurrentUser(r))
}
