package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/middlewares"
	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
	"github.com/satori/go.uuid"
)

type marketsImpl struct{}

func registerMarkets(router *httptreemux.TreeMux) {
	impl := &marketsImpl{}

	router.POST("/markets/:market/like", impl.like)
	router.POST("/markets/:market/dislike", impl.dislike)
	router.GET("/markets", impl.index)
	router.GET("/markets/:market", impl.market)
	router.GET("/markets/:market/candles/:granularity", impl.candles)
}

func (impl *marketsImpl) like(w http.ResponseWriter, r *http.Request, params map[string]string) {
	base, quote := getBaseQuote(params["market"])
	if base == "" || quote == "" {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
		return
	}
	if _, err := middlewares.CurrentUser(r).LikeMarket(r.Context(), base, quote); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderBlankResponse(w, r)
	}
}

func (impl *marketsImpl) dislike(w http.ResponseWriter, r *http.Request, params map[string]string) {
	base, quote := getBaseQuote(params["market"])
	if base == "" || quote == "" {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
		return
	}
	if err := middlewares.CurrentUser(r).DislikeMarket(r.Context(), base, quote); err != nil {
		views.RenderErrorResponse(w, r, err)
	} else {
		views.RenderBlankResponse(w, r)
	}
}

func (impl *marketsImpl) index(w http.ResponseWriter, r *http.Request, params map[string]string) {
	markets, err := models.ListActiveMarkets(r.Context(), middlewares.CurrentUser(r))
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}

	data := make([]map[string]interface{}, 0)
	for _, m := range markets {
		data = append(data, map[string]interface{}{
			"base":         m.Base,
			"quote":        m.Quote,
			"price":        number.FromString(fmt.Sprint(m.Price)).Persist(),
			"volume":       number.FromString(fmt.Sprint(m.Volume)).Round(2).Persist(),
			"total":        number.FromString(fmt.Sprint(m.Total)).Round(2).Persist(),
			"change":       number.FromString(fmt.Sprint(m.Change)).Persist(),
			"quote_usd":    fmt.Sprint(m.QuoteUSD),
			"base_symbol":  m.BaseSymbol(),
			"quote_symbol": m.QuoteSymbol(),
			"is_liked_by":  m.IsLikedBy,
		})
	}
	views.RenderDataResponse(w, r, data)
}

func (impl *marketsImpl) market(w http.ResponseWriter, r *http.Request, params map[string]string) {
	base, quote := getBaseQuote(params["market"])
	if base == "" || quote == "" {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
		return
	}

	m, err := models.GetMarket(r.Context(), base, quote)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}
	if m == nil {
		views.RenderErrorResponse(w, r, session.NotFoundError(r.Context()))
		return
	}

	data := map[string]interface{}{
		"base":         m.Base,
		"quote":        m.Quote,
		"price":        number.FromString(fmt.Sprint(m.Price)).Persist(),
		"volume":       number.FromString(fmt.Sprint(m.Volume)).Round(2).Persist(),
		"total":        number.FromString(fmt.Sprint(m.Total)).Round(2).Persist(),
		"change":       number.FromString(fmt.Sprint(m.Change)).Persist(),
		"quote_usd":    fmt.Sprint(m.QuoteUSD),
		"base_symbol":  m.BaseSymbol(),
		"quote_symbol": m.QuoteSymbol(),
	}
	views.RenderDataResponse(w, r, data)
}

func (impl *marketsImpl) candles(w http.ResponseWriter, r *http.Request, params map[string]string) {
	base, quote := getBaseQuote(params["market"])
	if base == "" || quote == "" {
		views.RenderDataResponse(w, r, []interface{}{})
		return
	}

	granularity, _ := strconv.Atoi(params["granularity"])
	candles, err := models.MarketCandles(r.Context(), base, quote, int64(granularity), 100)
	if err != nil {
		views.RenderErrorResponse(w, r, err)
		return
	}

	data := make([][]interface{}, 0)
	for _, c := range candles {
		data = append(data, []interface{}{
			c.Point, c.Low, c.High, c.Open, c.Close, c.Volume, c.Total,
		})
	}
	views.RenderDataResponse(w, r, data)
}

func getBaseQuote(market string) (string, string) {
	if len(market) != 73 {
		return "", ""
	}
	base := uuid.FromStringOrNil(market[0:36])
	if base.String() == uuid.Nil.String() {
		return "", ""
	}
	quote := uuid.FromStringOrNil(market[37:73])
	if quote.String() == uuid.Nil.String() {
		return "", ""
	}
	return base.String(), quote.String()
}
