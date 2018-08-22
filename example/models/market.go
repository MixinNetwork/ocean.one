package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

type Market struct {
	Base     string
	Quote    string
	Price    float64
	Volume   float64
	Total    float64
	Change   float64
	QuoteUSD float64

	IsLikedBy bool
}

var marketsColumnsFull = []string{"base", "quote", "price", "volume", "total", "change", "quote_usd"}

func (m *Market) valuesFull() []interface{} {
	return []interface{}{m.Base, m.Quote, m.Price, m.Volume, m.Total, m.Change, m.QuoteUSD}
}

func (m *Market) QuoteSymbol() string {
	return symbolsMap[m.Quote]
}

func (m *Market) BaseSymbol() string {
	return symbolsMap[m.Base]
}

func AllMarkets() []*Market {
	var markets []*Market
	for _, b := range usdtMarkets {
		markets = append(markets, &Market{Quote: "815b0b1a-2764-3736-8faa-42d694fa620a", Base: b})
	}
	for _, b := range btcMarkets {
		markets = append(markets, &Market{Quote: "c6d0c728-2624-429b-8e0d-d9d19b6592fa", Base: b})
	}
	for _, b := range xinMarkets {
		markets = append(markets, &Market{Quote: "c94ac88f-4671-3976-b60a-09064f1811e8", Base: b})
	}
	return markets
}

func GetMarket(ctx context.Context, base, quote string) (*Market, error) {
	it := session.Database(ctx).Query(ctx, spanner.Statement{
		SQL:    fmt.Sprintf("SELECT %s FROM markets WHERE base=@base AND quote=@quote", strings.Join(marketsColumnsFull, ",")),
		Params: map[string]interface{}{"base": base, "quote": quote},
	}, "markets", "GetMarket")
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	m, err := marketFromRow(row)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return m, nil
}

func ListActiveMarkets(ctx context.Context, user *User) ([]*Market, error) {
	inputs, err := ListMarkets(ctx, user)
	if err != nil {
		return inputs, err
	}

	var markets []*Market
	for _, m := range inputs {
		if m.Volume > 0 {
			markets = append(markets, m)
		}
	}
	return markets, nil
}

func ListMarkets(ctx context.Context, user *User) ([]*Market, error) {
	var markets []*Market
	favoritors, err := readFavoriteMarkets(ctx, user)
	if err != nil {
		return markets, nil
	}
	it := session.Database(ctx).Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf("SELECT %s FROM markets", strings.Join(marketsColumnsFull, ",")),
	}, "markets", "ListMarkets")
	defer it.Stop()

	for {
		row, err := it.Next()
		if err == iterator.Done {
			return markets, nil
		} else if err != nil {
			return nil, session.TransactionError(ctx, err)
		}
		m, err := marketFromRow(row)
		if err != nil {
			return nil, session.TransactionError(ctx, err)
		}
		m.IsLikedBy = favoritors[m.Base+m.Quote]
		markets = append(markets, m)
	}
}

func CreateOrUpdateMarket(ctx context.Context, base, quote string, price, volume, total, change, quoteUSD float64) error {
	for _, m := range AllMarkets() {
		if m.Base == base && m.Quote == quote {
			m.Price = price
			m.Volume = volume
			m.Total = total
			m.Change = change
			m.QuoteUSD = quoteUSD
			err := session.Database(ctx).Apply(ctx, []*spanner.Mutation{
				spanner.InsertOrUpdate("markets", marketsColumnsFull, m.valuesFull()),
			}, "markets", "INSERT", "CreateOrUpdateMarket")
			if err != nil {
				return session.TransactionError(ctx, err)
			}
			return nil
		}
	}
	return session.BadDataError(ctx)
}

func AggregateCandlesAsStats(ctx context.Context, base, quote string) (*Market, error) {
	market := &Market{Base: base, Quote: quote}
	start := time.Now().Add(-24 * time.Hour).Truncate(CandleGranularity15M).Unix()

	eit := session.Database(ctx).Query(ctx, spanner.Statement{
		SQL:    "SELECT point,close FROM candles WHERE base=@base AND quote=@quote AND granularity=@granularity ORDER BY base,quote,granularity,point DESC LIMIT 1",
		Params: map[string]interface{}{"base": base, "quote": quote, "granularity": CandleGranularity15M},
	}, "candles", "AggregateCandlesAsStats")
	defer eit.Stop()

	row, err := eit.Next()
	if err == iterator.Done {
		return market, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	var point int64
	var close float64
	err = row.Columns(&point, &close)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	market.Price = close
	if point < start {
		return market, nil
	}

	bit := session.Database(ctx).Query(ctx, spanner.Statement{
		SQL:    "SELECT point,close FROM candles WHERE base=@base AND quote=@quote AND granularity=@granularity AND point>=@point ORDER BY base,quote,granularity,point LIMIT 1",
		Params: map[string]interface{}{"base": base, "quote": quote, "granularity": CandleGranularity15M, "point": start},
	}, "candles", "AggregateCandlesAsStats")
	defer bit.Stop()

	row, err = bit.Next()
	if err == iterator.Done {
		return market, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	err = row.Columns(&point, &close)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	market.Change = (market.Price - close) / close

	sit := session.Database(ctx).Query(ctx, spanner.Statement{
		SQL:    "SELECT SUM(volume) AS volume, SUM(total) AS total FROM candles WHERE base=@base AND quote=@quote AND granularity=@granularity AND point>=@point",
		Params: map[string]interface{}{"base": base, "quote": quote, "granularity": CandleGranularity15M, "point": start},
	}, "candles", "AggregateCandlesAsStats")
	defer sit.Stop()

	row, err = sit.Next()
	if err == iterator.Done {
		return market, nil
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	var volume, total float64
	err = row.Columns(&volume, &total)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	market.Volume = volume
	market.Total = total
	return market, nil
}

func marketFromRow(row *spanner.Row) (*Market, error) {
	var m Market
	err := row.Columns(&m.Base, &m.Quote, &m.Price, &m.Volume, &m.Total, &m.Change, &m.QuoteUSD)
	return &m, err
}

var symbolsMap = map[string]string{
	"815b0b1a-2764-3736-8faa-42d694fa620a": "USDT",
	"c6d0c728-2624-429b-8e0d-d9d19b6592fa": "BTC",
	"fd11b6e3-0b87-41f1-a41f-f0e9b49e5bf0": "BCH",
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d": "EOS",
	"43d61dcd-e413-450d-80b8-101d5e903357": "ETH",
	"2204c1ee-0ea2-4add-bb9a-b3719cfff93a": "ETC",
	"76c802a2-7c88-447f-a93e-c29c9e5dd9c8": "LTC",
	"23dfb5a5-5d7b-48b6-905f-3970e3176e27": "XRP",
	"990c4c29-57e9-48f6-9819-7d986ea44985": "SC",
	"c94ac88f-4671-3976-b60a-09064f1811e8": "XIN",
}

var usdtMarkets = []string{
	"c6d0c728-2624-429b-8e0d-d9d19b6592fa", // BTC
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d", // EOS
	"43d61dcd-e413-450d-80b8-101d5e903357", // ETH
	"990c4c29-57e9-48f6-9819-7d986ea44985", // SC
	"c94ac88f-4671-3976-b60a-09064f1811e8", // XIN
}

var btcMarkets = []string{
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d", // EOS
	"43d61dcd-e413-450d-80b8-101d5e903357", // ETH
	"990c4c29-57e9-48f6-9819-7d986ea44985", // SC
	"c94ac88f-4671-3976-b60a-09064f1811e8", // XIN
}

var xinMarkets = []string{
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d", // EOS
	"43d61dcd-e413-450d-80b8-101d5e903357", // ETH
	"990c4c29-57e9-48f6-9819-7d986ea44985", // SC
	"43b645fc-a52c-38a3-8d3b-705e7aaefa15", // CANDY
}
