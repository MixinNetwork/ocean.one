package views

import (
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/session"
)

type ResponseView struct {
	Data  interface{} `json:"data,omitempty"`
	Error error       `json:"error,omitempty"`
	Prev  string      `json:"prev,omitempty"`
	Next  string      `json:"next,omitempty"`
}

func RenderDataResponse(w http.ResponseWriter, r *http.Request, view interface{}) {
	session.Render(r.Context()).JSON(w, http.StatusOK, ResponseView{Data: view})
}

func RenderErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	sessionError, ok := err.(session.Error)
	if !ok {
		sessionError = session.ServerError(r.Context(), err)
	}
	if sessionError.Code == 10001 {
		sessionError.Code = 500
	}
	session.Render(r.Context()).JSON(w, sessionError.Status, ResponseView{Error: sessionError})
}

func RenderBlankResponse(w http.ResponseWriter, r *http.Request) {
	session.Render(r.Context()).JSON(w, http.StatusOK, ResponseView{})
}
