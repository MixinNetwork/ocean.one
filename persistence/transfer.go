package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/go-number"
	"google.golang.org/api/iterator"
)

const (
	TransferSourceTradeConfirmed = "TRADE_CONFIRMED"
	TransferSourceOrderCancelled = "ORDER_CANCELLED"
	TransferSourceOrderFilled    = "ORDER_FILLED"
	TransferSourceOrderInvalid   = "ORDER_INVALID"
)

type Transfer struct {
	TransferId string    `spanner:"transfer_id"`
	Source     string    `spanner:"source"`
	Detail     string    `spanner:"detail"`
	AssetId    string    `spanner:"asset_id"`
	Amount     string    `spanner:"amount"`
	CreatedAt  time.Time `spanner:"created_at"`
	UserId     string    `spanner:"user_id"`
}

func ListPendingTransfers(ctx context.Context, limit int) ([]*Transfer, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	it := txn.Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf("SELECT transfer_id FROM transfers@{FORCE_INDEX=transfers_by_created} ORDER BY created_at LIMIT %d", limit),
	})
	defer it.Stop()

	transferIds := make([]string, 0)
	for {
		row, err := it.Next()
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
		transferIds = append(transferIds, id)
	}

	tit := txn.Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM transfers WHERE transfer_id IN UNNEST(@transfer_ids)",
		Params: map[string]interface{}{"transfer_ids": transferIds},
	})
	defer tit.Stop()

	transfers := make([]*Transfer, 0)
	for {
		row, err := tit.Next()
		if err == iterator.Done {
			return transfers, nil
		} else if err != nil {
			return transfers, err
		}
		var transfer Transfer
		err = row.ToStruct(&transfer)
		if err != nil {
			return transfers, err
		}
		transfers = append(transfers, &transfer)
	}
}

func ExpireTransfers(ctx context.Context, transfers []*Transfer) error {
	var set []spanner.KeySet
	for _, t := range transfers {
		set = append(set, spanner.Key{t.TransferId})
	}
	_, err := Spanner(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Delete("transfers", spanner.KeySets(set...)),
	})
	return err
}

func ReadTransferTrade(ctx context.Context, tradeId, assetId string) (*Trade, error) {
	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM trades WHERE trade_id=@trade_id",
		Params: map[string]interface{}{"trade_id": tradeId},
	})
	defer it.Stop()

	for {
		row, err := it.Next()
		if err == iterator.Done {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		var trade Trade
		err = row.ToStruct(&trade)
		if err != nil {
			return nil, err
		}
		if trade.FeeAssetId == assetId {
			return &trade, nil
		}
	}
}

func CreateRefundTransfer(ctx context.Context, userId, assetId string, amount number.Decimal, trace string) error {
	if amount.Exhausted() {
		return nil
	}
	transfer := &Transfer{
		TransferId: getSettlementId(trace, "REFUND"),
		Source:     TransferSourceOrderInvalid,
		Detail:     trace,
		AssetId:    assetId,
		Amount:     amount.Persist(),
		CreatedAt:  time.Now(),
		UserId:     userId,
	}
	mutation, err := spanner.InsertStruct("transfers", transfer)
	if err != nil {
		return err
	}
	_, err = Spanner(ctx).Apply(ctx, []*spanner.Mutation{mutation})
	return err
}
