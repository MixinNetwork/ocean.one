package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

func MarketTrades(ctx context.Context, market string, offset time.Time, limit int) ([]*Trade, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	if limit > 100 {
		limit = 100
	}

	base, quote := getBaseQuote(market)
	if base == "" || quote == "" {
		return nil, nil
	}

	query := "SELECT trade_id FROM trades@{FORCE_INDEX=trades_by_base_quote_created_desc} WHERE base=@base AND quote=@quote AND created_at<@offset"
	query = query + " ORDER BY base,quote,created_at DESC"
	query = fmt.Sprint("%s LIMIT %d", query, limit)
	params := map[string]interface{}{"base": base, "quote": quote, "offset": offset}

	iit := txn.Query(ctx, spanner.Statement{query, params})
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
		SQL:    "SELECT * FROM trades WHERE trade_id IN UNNEST(@trade_ids)",
		Params: map[string]interface{}{"trade_ids": tradeIds},
	})
	defer tit.Stop()

	var trades []*Trade
	for {
		row, err := tit.Next()
		if err == iterator.Done {
			return trades, nil
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
