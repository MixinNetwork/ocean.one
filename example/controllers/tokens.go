package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/middlewares"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
)

type tokensImpl struct{}

type tokenRequest struct {
	Category string `json:"category"`
	Method   string `json:"method"`
	URI      string `json:"uri"`
}

func registerTokens(router *httptreemux.TreeMux) {
	impl := &tokensImpl{}

	router.POST("/tokens", impl.create)
}

func (impl *tokensImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	key := middlewares.CurrentUser(r).Key
	switch body.Category {
	case "MIXIN":
		token, err := key.MixinToken(r.Context(), body.Method, body.URI)
		if err != nil {
			views.RenderErrorResponse(w, r, err)
		} else {
			views.RenderDataResponse(w, r, map[string]string{"token": token})
		}
	case "OCEAN":
		token, err := key.OceanToken(r.Context())
		if err != nil {
			views.RenderErrorResponse(w, r, err)
		} else {
			views.RenderDataResponse(w, r, map[string]string{"token": token})
		}
	default:
		views.RenderErrorResponse(w, r, session.BadDataError(r.Context()))
	}
}
