package controllers

import (
	"net/http"
	"strconv"

	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/views"
	"github.com/dimfeld/httptreemux"
	"github.com/satori/go.uuid"
)

type marketsImpl struct{}

func registerMarkets(router *httptreemux.TreeMux) {
	impl := &marketsImpl{}

	router.GET("/markets/:market/candles/:granularity", impl.candles)
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
