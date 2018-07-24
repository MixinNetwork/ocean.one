package middlewares

import (
	"net/http"

	"github.com/newrelic/go-agent"
)

func NewRelic(handler http.Handler, app newrelic.Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn := app.StartTransaction(r.Method+" "+r.URL.Path, w, r)
		defer txn.End()

		handler.ServeHTTP(txn, r)
	})
}
