package main

import (
	"fmt"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/controllers"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/middlewares"
	"github.com/bugsnag/bugsnag-go"
	"github.com/dimfeld/httptreemux"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gorilla/handlers"
	"github.com/unrolled/render"
)

func StartServer(spanner *spanner.Client) error {
	logger, err := durable.NewLoggerClient(config.GoogleCloudProject, config.Environment != "production")
	if err != nil {
		return err
	}
	defer logger.Close()

	limiter, err := durable.NewLimiter(config.RedisRateLimiterAddress, config.RedisRateLimiterDatabase)
	if err != nil {
		return err
	}

	router := httptreemux.New()
	controllers.RegisterHanders(router)
	controllers.RegisterRoutes(router)
	handler := middlewares.Authenticate(router)
	handler = middlewares.Limit(handler)
	handler = middlewares.Constraint(handler)
	handler = middlewares.Context(handler, spanner, limiter, render.New(render.Options{UnEscapeHTML: true}))
	handler = middlewares.NewRelic(handler, setupNewRelic("http"))
	handler = middlewares.Stats(handler, "http", config.HTTPLogRequestBody, config.BuildVersion)
	handler = middlewares.Log(handler, logger, "http")
	handler = handlers.ProxyHeaders(handler)
	handler = bugsnag.Handler(handler)

	return gracehttp.Serve(&http.Server{Addr: fmt.Sprintf(":%d", config.HTTPListenPort), Handler: handler})
}
