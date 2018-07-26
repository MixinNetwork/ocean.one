package controllers

import (
	"net/http"
	"runtime"

	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
)

func RegisterRoutes(router *httptreemux.TreeMux) {
	router.GET("/", root)
	router.GET("/_hc", healthCheck)

	registerVerifications(router)
}

func root(w http.ResponseWriter, r *http.Request, params map[string]string) {
	views.RenderDataResponse(w, r, map[string]string{
		"build":      config.BuildVersion + "-" + runtime.Version(),
		"developers": "https://github.com/MixinNetwork/ocean.one/example",
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request, params map[string]string) {
	views.RenderBlankResponse(w, r)
}
