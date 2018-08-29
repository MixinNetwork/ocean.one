package middlewares

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
)

func parseRemoteAddr(remoteAddress string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	}
	ip := net.ParseIP(remoteAddress)
	if ip == nil {
		return "", fmt.Errorf("invalid remote address %s", remoteAddress)
	}
	return ip.String(), nil
}

func Constraint(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && !strings.HasPrefix(strings.ToLower(r.URL.Path), "/callbacks/twilio/") {
			views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
			return
		}

		if strings.HasPrefix(r.UserAgent(), "Mixin/0.1.") || strings.HasPrefix(r.UserAgent(), "Mixin/0.2.") || strings.HasPrefix(r.UserAgent(), "Mixin/0.3.") {
			views.RenderErrorResponse(w, r, session.AuthorizationError(r.Context()))
			return
		}

		remoteAddress, err := parseRemoteAddr(r.RemoteAddr)
		if err != nil {
			views.RenderBlankResponse(w, r)
			return
		}
		if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
			cfIP, err := parseRemoteAddr(ip)
			if err != nil {
				views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
				return
			}
			if remoteAddress != cfIP {
				views.RenderErrorResponse(w, r, session.BadRequestError(r.Context()))
				return
			}
		}

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Mixin-Conversation-ID")
			w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,DELETE")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == "OPTIONS" {
			views.RenderBlankResponse(w, r)
		} else {
			ctx := session.WithRemoteAddress(r.Context(), remoteAddress)
			handler.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
