package controllers

import (
	"fmt"
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/bugsnag/bugsnag-go/errors"
	"github.com/dimfeld/httptreemux"
)

func RegisterHanders(router *httptreemux.TreeMux) {
	router.MethodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request, _ map[string]httptreemux.HandlerFunc) {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	}
	router.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
	}
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, rcv interface{}) {
		err := fmt.Errorf(string(errors.New(rcv, 2).Stack()))
		views.RenderErrorResponse(w, r, session.ServerError(r.Context(), err))
	}
}
