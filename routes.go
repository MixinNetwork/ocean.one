package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MixinMessenger/ocean.one/cache"
	"github.com/MixinMessenger/ocean.one/persistence"
	"github.com/bugsnag/bugsnag-go/errors"
	"github.com/dimfeld/httptreemux"
	"github.com/unrolled/render"
)

type R struct{}

func NewRouter() *httptreemux.TreeMux {
	router, impl := httptreemux.New(), &R{}
	router.GET("/markets/:id/ticker", impl.marketTicker)
	router.GET("/markets/:id/book", impl.marketBook)
	router.GET("/markets/:id/trades", impl.marketTrades)
	router.GET("/markets/:id/candles", impl.marketCandles)
	router.GET("/orders", impl.orders)
	registerHanders(router)
	return router
}

func (impl *R) marketTicker(w http.ResponseWriter, r *http.Request, params map[string]string) {
}

func (impl *R) marketBook(w http.ResponseWriter, r *http.Request, params map[string]string) {
	book, err := cache.Book(r.Context(), params["id"])
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	} else {
		render.New().JSON(w, http.StatusOK, book)
	}
}

func (impl *R) marketTrades(w http.ResponseWriter, r *http.Request, params map[string]string) {
	trades, err := persistence.MarketTrades(r.Context(), params["id"], time.Now(), 100)
	if err != nil {
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	data := make([]map[string]interface{}, 0)
	for _, t := range trades {
		data = append(data, map[string]interface{}{
			"trade_id":   t.TradeId,
			"liquidity":  t.Liquidity,
			"market":     params["id"],
			"side":       t.Side,
			"price":      t.Price,
			"amount":     t.Amount,
			"created_at": t.CreatedAt,
		})
	}
	render.New().JSON(w, http.StatusOK, data)
}

func (impl *R) marketCandles(w http.ResponseWriter, r *http.Request, params map[string]string) {
}

func (impl *R) orders(w http.ResponseWriter, r *http.Request, params map[string]string) {
}

func registerHanders(router *httptreemux.TreeMux) {
	router.MethodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request, _ map[string]httptreemux.HandlerFunc) {
		render.New().JSON(w, http.StatusNotFound, map[string]interface{}{})
	}
	router.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		render.New().JSON(w, http.StatusNotFound, map[string]interface{}{})
	}
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, rcv interface{}) {
		err := fmt.Errorf(string(errors.New(rcv, 2).Stack()))
		render.New().JSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}
}
