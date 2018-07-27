package middlewares

import (
	"errors"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/bugsnag/bugsnag-go"
)

func getRareRoutePattern(r *http.Request) string {
	var rareRoutes = [...][2]string{
		{"POST", "^/verifications"},
	}

	path := strings.ToLower(path.Clean(r.URL.Path))
	for _, route := range rareRoutes {
		if route[0] != r.Method {
			continue
		}
		if matched, _ := regexp.MatchString(route[1], path); matched {
			return route[0] + route[1]
		}
	}

	return ""
}

func checkLimiter(r *http.Request) error {
	limiter := session.Limiter(r.Context())
	var remoteAddr = session.RemoteAddress(r.Context())
	if strings.TrimSpace(remoteAddr) == "" {
		return errors.New(strings.TrimSpace("rate limit without valid IP address " + remoteAddr))
	}

	key := "limiter:general:ip:" + remoteAddr
	if count, err := limiter.Available(key, time.Second*5, 100000, true); err != nil {
		bugsnag.Notify(err, r)
	} else if count < 1 {
		return errors.New("general rate limit error 100000r/5s")
	}
	if count, err := limiter.Available(key, time.Hour, 10000000, true); err != nil {
		bugsnag.Notify(err, r)
	} else if count < 1 {
		return errors.New("general rate limit error 10000000r/h")
	}

	if pattern := getRareRoutePattern(r); pattern != "" {
		keys := []string{"limiter:rare:ip:" + remoteAddr + ":" + pattern}
		for _, key := range keys {
			if count, err := limiter.Available(key, 10*time.Minute, 10, true); err != nil {
				bugsnag.Notify(err, r)
			} else if count < 1 {
				return errors.New("rare rate limit error 10r/10m")
			}
			if count, err := limiter.Available(key, 24*time.Hour, 30, true); err != nil {
				bugsnag.Notify(err, r)
			} else if count < 1 {
				return errors.New("rare rate limit error 30r/d")
			}
		}
	}
	return nil
}

func Limit(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_hc" {
			handler.ServeHTTP(w, r)
			return
		}
		if err := checkLimiter(r); err != nil {
			bugsnag.Notify(err, r)
			views.RenderErrorResponse(w, r, session.TooManyRequestsError(r.Context()))
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}
