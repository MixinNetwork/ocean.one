package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/middlewares"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
)

type withdrawalsImpl struct{}

type withdrawalRequest struct {
	TraceId string
	AssetId string
	Amount  string
	Memo    string
}

func registerWithdrawals(router *httptreemux.TreeMux) {
	impl := &withdrawalsImpl{}

	router.POST("/withdrawals", impl.create)
}

func (impl *withdrawalsImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body withdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	err := middlewares.CurrentUser(r).CreateWithdrawal(r.Context(), body.AssetId, number.FromString(body.Amount), body.TraceId, body.Memo)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderBlankResponse(w, r)
	}
}
