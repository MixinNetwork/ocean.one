package main

import (
	"context"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/MixinNetwork/ocean.one/cache"
	"github.com/MixinNetwork/ocean.one/config"
	"github.com/MixinNetwork/ocean.one/persistence"
	"github.com/bugsnag/bugsnag-go"
	"github.com/dimfeld/httptreemux"
	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"github.com/unrolled/render"
)

type RequestHandler struct {
	hub      *cache.Hub
	upgrader *websocket.Upgrader
	router   *httptreemux.TreeMux
}

func (handler *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer bugsnag.Recover(r, bugsnag.ErrorClass{Name: "cache.ServeHTTP"})

	if r.URL.Path == "/_hc" {
		render.New().JSON(w, http.StatusOK, map[string]interface{}{})
		return
	}

	if r.URL.Path != "/" {
		handler.router.ServeHTTP(w, r)
		return
	}

	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		cp, err := persistence.ReadPropertyAsTime(r.Context(), CheckpointMixinNetworkSnapshots)
		if err != nil {
			render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		ac, err := persistence.CountPendingActions(r.Context())
		if err != nil {
			render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		tc, err := persistence.CountPendingTransfers(r.Context())
		if err != nil {
			render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		data := map[string]interface{}{
			"build":      config.BuildVersion + "-" + runtime.Version(),
			"developers": "https://github.com/MixinNetwork/ocean.one",
			"checkpoint": cp,
			"actions":    ac,
			"transfers":  tc,
		}
		render.New().JSON(w, http.StatusOK, map[string]interface{}{"data": data})
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
	client, err := cache.NewClient(ctx, handler.hub, conn, cid.String(), cancel)
	if err != nil {
		return
	}
	if err := handler.hub.Register(ctx, client); err != nil {
		return
	}
	defer handler.hub.Unregister(client)
	go client.WritePump(ctx)
	client.ReadPump(ctx)
}

func StartHTTP(ctx context.Context) error {
	hub := cache.NewHub()
	go hub.Run(ctx)

	rh := &RequestHandler{
		hub: hub,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: 60 * time.Second,
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			CheckOrigin:      func(r *http.Request) bool { return true },
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				render.New().JSON(w, status, map[string]interface{}{"error": reason.Error()})
			},
		},
		router: NewRouter(),
	}
	handler := handleContext(rh, ctx)
	handler = handleCORS(handler)
	handler = handlers.ProxyHeaders(handler)
	handler = bugsnag.Handler(handler)

	server := &http.Server{Addr: ":7000", Handler: handler}
	return server.ListenAndServe()
}

func handleContext(handler http.Handler, src context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := cache.SetupRedis(r.Context(), cache.Redis(src))
		ctx = persistence.SetupSpanner(ctx, persistence.Spanner(src))
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
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
