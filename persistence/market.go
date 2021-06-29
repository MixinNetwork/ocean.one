package persistence

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/gofrs/uuid"
	"google.golang.org/api/iterator"
)

func LastTrade(ctx context.Context, market string) (*Trade, error) {
	base, quote := getBaseQuote(market)
	if base == "" || quote == "" {
		return nil, nil
	}

	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM trades@{FORCE_INDEX=trades_by_base_quote_created_desc} WHERE base_asset_id=@base AND quote_asset_id=@quote ORDER BY base_asset_id,quote_asset_id,created_at DESC",
		Params: map[string]interface{}{"base": base, "quote": quote},
	})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var t Trade
	err = row.ToStruct(&t)
	return &t, err
}

func MarketTrades(ctx context.Context, market string, offset time.Time, order string, limit int) ([]*Trade, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	if limit > 100 {
		limit = 100
	}
	cmp := "<"
	if order != "DESC" {
		order = "ASC"
		cmp = ">"
	}

	base, quote := getBaseQuote(market)
	if base == "" || quote == "" {
		return nil, nil
	}

	query := "SELECT trade_id FROM trades@{FORCE_INDEX=trades_by_base_quote_created_%s} WHERE base_asset_id=@base AND quote_asset_id=@quote AND created_at%s=@offset AND liquidity=@liquidity"
	query = fmt.Sprintf(query, strings.ToLower(order), cmp)
	query = query + " ORDER BY base_asset_id,quote_asset_id,created_at " + order
	query = fmt.Sprintf("%s LIMIT %d", query, limit)
	params := map[string]interface{}{"base": base, "quote": quote, "offset": offset, "liquidity": TradeLiquidityMaker}

	iit := txn.Query(ctx, spanner.Statement{SQL: query, Params: params})
	defer iit.Stop()

	var tradeIds []string
	for {
		row, err := iit.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		var id string
		err = row.Columns(&id)
		if err != nil {
			return nil, err
		}
		tradeIds = append(tradeIds, id)
	}

	tit := txn.Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM trades WHERE trade_id IN UNNEST(@trade_ids) AND liquidity=@liquidity",
		Params: map[string]interface{}{"trade_ids": tradeIds, "liquidity": TradeLiquidityMaker},
	})
	defer tit.Stop()

	var trades []*Trade
	for {
		row, err := tit.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return trades, err
		}
		var t Trade
		err = row.ToStruct(&t)
		if err != nil {
			return trades, err
		}
		trades = append(trades, &t)
	}
	if order == "DESC" {
		sort.Slice(trades, func(i, j int) bool { return trades[i].CreatedAt.After(trades[j].CreatedAt) })
	} else {
		sort.Slice(trades, func(i, j int) bool { return trades[i].CreatedAt.Before(trades[j].CreatedAt) })
	}
	return trades, nil
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
