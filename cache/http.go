package cache

import (
	"context"
	"net/http"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"github.com/unrolled/render"
)

type RequestHandler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

func (handler *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer bugsnag.Recover(r, bugsnag.ErrorClass{Name: "cache.ServeHTTP"})

	if r.URL.Path == "/_hc" {
		render.New().JSON(w, http.StatusOK, map[string]interface{}{})
		return
	}

	if r.URL.Path != "/" {
		render.New().JSON(w, http.StatusNotFound, map[string]interface{}{})
		return
	}

	conn, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	cid, err := uuid.NewV4()
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	client, err := NewClient(ctx, handler.hub, conn, cid.String(), cancel)
	if err != nil {
		return
	}
	if err := handler.hub.Register(r.Context(), client); err != nil {
		return
	}
	defer handler.hub.Unregister(client)
	go client.WritePump(ctx)
	client.ReadPump(ctx)
}

func StartHTTP(ctx context.Context) error {
	hub := NewHub()
	go hub.Run(ctx)

	rh := &RequestHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 60 * time.Second,
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			CheckOrigin:      func(r *http.Request) bool { return true },
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				render.New().JSON(w, status, map[string]interface{}{"error": reason.Error()})
			},
		},
	}
	handler := handleCORS(rh)
	handler = handlers.ProxyHeaders(handler)
	handler = bugsnag.Handler(handler)

	server := &http.Server{Addr: ":7000", Handler: handler}
	return server.ListenAndServe()
}

func handleCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			handler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,Authorization,Mixin-Conversation-ID")
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,DELETE")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == "OPTIONS" {
			render.New().JSON(w, http.StatusOK, map[string]interface{}{})
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}
