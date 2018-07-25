package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
)

type verificationsImpl struct{}

type verificationRequest struct {
	Category          string `json:"category"`
	Receiver          string `json:"receiver"`
	RecaptchaResponse string `json:"recaptcha_response"`
	Code              string `json:"code"`
}

func registerVerifications(router *httptreemux.TreeMux) {
	impl := &verificationsImpl{}

	router.POST("/verifications", impl.create)
	router.POST("/verifications/:id", impl.verify)
}

func (impl *verificationsImpl) create(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	var body verificationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	pv, err := models.CreateVerification(r.Context(), body.Category, body.Receiver, body.RecaptchaResponse)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}

	views.RenderDataResponse(w, r, map[string]interface{}{
		"type":            "verification",
		"verification_id": pv.VerificationId,
		"is_verified":     false,
	})
}

func (impl *verificationsImpl) verify(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var body verificationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
		return
	}

	pv, err := models.DoVerification(r.Context(), params["id"], body.Code)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}

	views.RenderDataResponse(w, r, map[string]interface{}{
		"type":            "verification",
		"verification_id": pv.VerificationId,
		"is_verified":     pv.VerifiedAt.Valid,
	})
}
