package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

const (
	CandleGranularity1M  = 60
	CandleGranularity5M  = 300
	CandleGranularity15M = 900
	CandleGranularity1H  = 3600
	CandleGranularity6H  = 21600
	CandleGranularity1D  = 86400
)

type Candle struct {
	Base        string
	Quote       string
	Granularity int64
	Point       int64
	Open        float64
	Close       float64
	High        float64
	Low         float64
	Volume      float64
	Total       float64
}

var candlesColumnsFull = []string{"base", "quote", "granularity", "point", "open", "close", "high", "low", "volume", "total"}

func (c *Candle) valuesFull() []interface{} {
	return []interface{}{c.Base, c.Quote, c.Granularity, c.Point, c.Open, c.Close, c.High, c.Low, c.Volume, c.Total}
}

func MarketCandles(ctx context.Context, base, quote string, granularity, limit int64) ([]*Candle, error) {
	if granularity != CandleGranularity1M && granularity != CandleGranularity5M &&
		granularity != CandleGranularity15M && granularity != CandleGranularity1H &&
		granularity != CandleGranularity6H && granularity != CandleGranularity1D {
		return make([]*Candle, 0), nil
	}

	query := "SELECT %s FROM candles WHERE base=@base AND quote=@quote AND granularity=@granularity ORDER BY base,quote,granularity,point DESC LIMIT %d"
	it := session.Database(ctx).Query(ctx, spanner.Statement{
		SQL:    fmt.Sprintf(query, strings.Join(candlesColumnsFull, ","), limit),
		Params: map[string]interface{}{"base": base, "quote": quote, "granularity": granularity},
	}, "candles", "MarketCandles")
	defer it.Stop()

	inputs := make([]*Candle, 0)
	filter := make(map[int64]*Candle)
	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, session.TransactionError(ctx, err)
		}
		c, err := candleFromRow(row)
		if err != nil {
			return nil, session.TransactionError(ctx, err)
		}
		inputs = append(inputs, c)
		filter[c.Point] = c
	}
	if len(inputs) == 0 {
		return inputs, nil
	}

	end := inputs[0].Point
	start := end - (limit-1)*granularity
	for _, c := range inputs {
		if c.Point > start {
			continue
		}
		if c.Point == start {
			break
		}
		filter[start] = c.copyAsEmpty(start)
		break
	}
	if filter[start] == nil {
		start = inputs[len(inputs)-1].Point
	}

	var candles []*Candle
	for i := start; i <= end; i += granularity {
		c := filter[i]
		if c != nil {
			candles = append(candles, c)
			continue
		}
		c = candles[len(candles)-1]
		candles = append(candles, c.copyAsEmpty(i))
	}

	return candles, nil
}

func (c *Candle) copyAsEmpty(point int64) *Candle {
	return &Candle{
		Base:        c.Base,
		Quote:       c.Quote,
		Granularity: c.Granularity,
		Point:       point,
		Open:        c.Close,
		Close:       c.Close,
		High:        c.Close,
		Low:         c.Close,
		Volume:      0,
		Total:       0,
	}
}

func CreateOrUpdateCandle(ctx context.Context, base, quote string, price, amount number.Decimal, createdAt time.Time) error {
	var keys []spanner.KeySet
	var candles = make(map[string]*Candle)

	for _, g := range []int64{
		CandleGranularity1M,
		CandleGranularity5M,
		CandleGranularity15M,
		CandleGranularity1H,
		CandleGranularity6H,
		CandleGranularity1D,
	} {
		p := createdAt.UTC().Truncate(time.Duration(g) * time.Second).Unix()
		keys = append(keys, spanner.Key{base, quote, g, p})
		candles[fmt.Sprintf("%d:%d", g, p)] = &Candle{
			Base:        base,
			Quote:       quote,
			Granularity: g,
			Point:       p,
			Open:        price.Float64(),
			Close:       price.Float64(),
			High:        price.Float64(),
			Low:         price.Float64(),
			Volume:      amount.Float64(),
			Total:       price.Mul(amount).Float64(),
		}
	}

	it := session.Database(ctx).Read(ctx, "candles", spanner.KeySets(keys...), candlesColumnsFull, "CreateOrUpdateCandle")
	defer it.Stop()

	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return session.TransactionError(ctx, err)
		}
		c, err := candleFromRow(row)
		if err != nil {
			return session.TransactionError(ctx, err)
		}
		n := candles[fmt.Sprintf("%d:%d", c.Granularity, c.Point)]
		n.Open = c.Open
		if c.High > n.High {
			n.High = c.High
		}
		if c.Low < n.Low {
			n.Low = c.Low
		}
		n.Volume = n.Volume + c.Volume
		n.Total = n.Total + c.Total
	}

	var mutations []*spanner.Mutation
	for _, c := range candles {
		mutations = append(mutations, spanner.InsertOrUpdate("candles", candlesColumnsFull, c.valuesFull()))
	}
	err := session.Database(ctx).Apply(ctx, mutations, "candles", "INSERT", "CreateOrUpdateCandle")
	if err != nil {
		return session.TransactionError(ctx, err)
	}
	return nil
}

func candleFromRow(row *spanner.Row) (*Candle, error) {
	var c Candle
	err := row.Columns(&c.Base, &c.Quote, &c.Granularity, &c.Point, &c.Open, &c.Close, &c.High, &c.Low, &c.Volume, &c.Total)
	return &c, err
}
