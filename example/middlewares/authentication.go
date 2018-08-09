package middlewares

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
)

var whitelist = [][2]string{
	{"GET", "^/$"},
	{"GET", "^/_hc$"},
	{"GET", "^/markets"},
	{"POST", "^/verifications"},
	{"POST", "^/users$"},
	{"POST", "^/passwords$"},
	{"POST", "^/sessions$"},
	{"POST", "^/callbacks/twilio/"},
}

type contextValueKey struct{ int }

var keyCurrentUser = contextValueKey{1000}

func CurrentUser(r *http.Request) *models.User {
	user, _ := r.Context().Value(keyCurrentUser).(*models.User)
	return user
}

func Authenticate(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			handleUnauthorized(handler, w, r)
			return
		}
		r = r.WithContext(session.WithAuthorizationInfo(r.Context(), header[7:]))

		claims := session.AuthorizationInfo(r.Context())
		content := r.Method + r.RequestURI + session.RequestBody(r.Context())
		sum := sha256.Sum256([]byte(content))
		if claims["sig"] != hex.EncodeToString(sum[:]) {
			handleUnauthorized(handler, w, r)
			return
		}

		user, err := models.AuthenticateWithToken(r.Context(), header[7:])
		if err != nil {
			views.RenderErrorResponse(w, r, err)
		} else if user == nil {
			handleUnauthorized(handler, w, r)
		} else {
			ctx := context.WithValue(r.Context(), keyCurrentUser, user)
			handler.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func handleUnauthorized(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	for _, pp := range whitelist {
		if pp[0] != r.Method {
			continue
		}
		if matched, _ := regexp.MatchString(pp[1], strings.ToLower(r.URL.Path)); matched {
			handler.ServeHTTP(w, r)
			return
		}
	}

	views.RenderErrorResponse(w, r, session.AuthorizationError(r.Context()))
}
