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

type ordersImpl struct{}

func registerOrders(router *httptreemux.TreeMux) {
	impl := &ordersImpl{}

	router.POST("/orders", impl.create)
	router.POST("/orders/:id/cancel", impl.cancel)
}

func (impl *ordersImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body models.OrderAction
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	err := middlewares.CurrentUser(r).CreateOrder(r.Context(), &body)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderBlankResponse(w, r)
	}
}

func (impl *ordersImpl) cancel(w http.ResponseWriter, r *http.Request, params map[string]string) {
	err := middlewares.CurrentUser(r).CancelOrder(r.Context(), params["id"])
	if err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderBlankResponse(w, r)
	}
}
