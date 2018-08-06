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
	FullName       string `json:"full_name"`
}

func registerUsers(router *httptreemux.TreeMux) {
	impl := &usersImpl{}

	router.POST("/users", impl.create)
	router.GET("/me", impl.me)
	router.POST("/me", impl.update)
	router.POST("/me/mixin", impl.mixin)
}

func (impl *usersImpl) mixin(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
	} else if user, err := middlewares.CurrentUser(r).ConnectMixin(r.Context(), body.Code); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderUserWithAuthentication(w, r, user)
	}
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

func (impl *usersImpl) update(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body userRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	user, err := middlewares.CurrentUser(r).UpdateName(r.Context(), body.FullName)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}
	views.RenderUserWithAuthentication(w, r, user)
}
