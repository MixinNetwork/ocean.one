package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
)

type Trade struct {
	Base      string    `json:"base"`
	Quote     string    `json:"quote"`
	Amount    string    `json:"amount"`
	Price     string    `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	TradeId   string    `json:"trade_id"`
}

type CandleService struct{}

func (service *CandleService) Healthy(ctx context.Context) bool {
	if err := standardServiceHealth(ctx); err != nil {
		return false
	}
	return true
}

func (service *CandleService) Run(ctx context.Context) error {
	if err := standardServiceHealth(ctx); err != nil {
		return err
	}
	for _, m := range models.AllMarkets() {
		go service.handleMarketCandles(ctx, m.Base, m.Quote)
		go service.handleMarketStats(ctx, m.Base, m.Quote)
	}
	for {
		err := standardServiceHealth(ctx)
		if err != nil {
			session.ServerError(ctx, err)
		}
		time.Sleep(1 * time.Second)
	}
}

func (service *CandleService) handleMarketStats(ctx context.Context, base, quote string) {
	const interval = 500
	for {
		time.Sleep(interval)
		m, err := models.AggregateCandlesAsStats(ctx, base, quote)
		if err != nil {
			session.ServerError(ctx, err)
			continue
		}

		err = models.CreateOrUpdateMarket(ctx, m.Base, m.Quote, m.Price, m.Volume, m.Total, m.Change)
		if err != nil {
			session.ServerError(ctx, err)
		}
	}
}

func (service *CandleService) handleMarketCandles(ctx context.Context, base, quote string) {
	const limit = 100
	const interval = 500
	var key = fmt.Sprintf("candles-checkpoint-%s-%s", base, quote)
	var cache = make(map[string]bool)

	for {
		checkpoint, err := models.ReadPropertyAsTime(ctx, key)
		if err != nil {
			session.ServerError(ctx, err)
			time.Sleep(interval)
			continue
		}
		trades, err := fetchTrades(ctx, base, quote, checkpoint, limit)
		if err != nil {
			session.ServerError(ctx, err)
			time.Sleep(interval)
			continue
		}

		for _, t := range trades {
			if cache[t.TradeId] {
				continue
			}
			if t.Quote != quote || t.Base != base {
				log.Panicln(base, quote, t)
			}

			for {
				err := models.CreateOrUpdateCandle(ctx, t.Base, t.Quote, number.FromString(t.Price), number.FromString(t.Amount), t.CreatedAt)
				if err == nil {
					break
				}
				session.ServerError(ctx, err)
				time.Sleep(interval)
			}

			checkpoint = t.CreatedAt
			cache[t.TradeId] = true
		}
		err = models.WriteTimeProperty(ctx, key, checkpoint)
		if err != nil {
			session.ServerError(ctx, err)
		}
		if len(trades) < limit {
			time.Sleep(interval)
		}
	}
}

var httpClient *http.Client

func fetchTrades(ctx context.Context, base, quote string, offset time.Time, limit int) ([]*Trade, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	path := "https://events.ocean.one/markets/%s-%s/trades?order=ASC&limit=%d&offset=%s"
	path = fmt.Sprintf(path, base, quote, limit, offset.UTC().Format(time.RFC3339Nano))

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	req.Close = true
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	defer resp.Body.Close()

	var body struct {
		Data  []*Trade `json:"data"`
		Error error    `json:"error"`
	}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}

	if body.Error != nil {
		return nil, session.ServerError(ctx, body.Error)
	}
	return body.Data, nil
}
