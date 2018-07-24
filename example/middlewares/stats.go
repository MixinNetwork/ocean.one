package middlewares

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime"
	"time"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
)

func Stats(handler http.Handler, service string, logRequestBody bool, buildVersion string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startAt := time.Now()

		if r.ContentLength > 0 && r.Body != nil {
			p, err := ioutil.ReadAll(r.Body)
			if err != nil {
				views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
				return
			}
			r.Body.Close()
			r.Body = ioutil.NopCloser(bytes.NewBuffer(p))
			r = r.WithContext(session.WithRequestBody(r.Context(), string(p)))
			if logRequestBody {
				session.Logger(r.Context()).Info(string(p))
			}
		}

		if service == "blaze" {
			handler.ServeHTTP(w, r)
			spent := time.Now().Sub(startAt).Seconds()
			session.Logger(r.Context()).Infof("{%s %s FINISHED IN %f seconds}", r.Method, r.URL, spent)
		} else {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)

			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			spent := time.Now().Sub(startAt)
			w.Header().Set("X-Build-Info", buildVersion+"-"+runtime.Version())
			w.Header().Set("X-Request-Id", r.Header.Get("X-Request-Id"))
			w.Header().Set("X-Runtime", fmt.Sprintf("%f", spent.Seconds()))
			w.WriteHeader(rec.Code)
			contentLength, _ := rec.Body.WriteTo(w)
			session.Logger(r.Context()).FillResponse(rec.Code, contentLength, spent)
			session.Logger(r.Context()).Infof("{%s %s RESPOND %d bytes FINISHED %d IN %f seconds}", r.Method, r.URL, contentLength, rec.Code, spent.Seconds())
		}
	})
}
