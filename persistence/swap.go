package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	SwapAdd    = "SWAP_ADD"
	SwapTrade  = "SWAP_TRADE"
	SwapRemove = "SWAP_REMOVE"
)

type SwapAction struct {
	ActionId     string    `spanner:"action_id"`
	PoolId       string    `spanner:"pool_id"`
	BaseAssetId  string    `spanner:"base_asset_id"`
	QuoteAssetId string    `spanner:"quote_asset_id"`
	Action       string    `spanner:"action"`
	AssetId      string    `spanner:"asset_id"`
	Amount       string    `spanner:"amount"`
	BrokerId     string    `spanner:"broker_id"`
	UserId       string    `spanner:"user_id"`
	TraceId      string    `spanner:"trace_id"`
	CreatedAt    time.Time `spanner:"created_at"`
}

func (a *SwapAction) Key() string {
	return a.BaseAssetId + "-" + a.QuoteAssetId
}

func ListPendingSwapActions(ctx context.Context, checkpoint time.Time, limit int) ([]*SwapAction, error) {
	txn := Spanner(ctx).Single()
	defer txn.Close()

	it := txn.Query(ctx, spanner.Statement{
		SQL:    fmt.Sprintf("SELECT * FROM swap_actions@{FORCE_INDEX=swap_actions_by_created} WHERE created_at>=@checkpoint ORDER BY created_at LIMIT %d", limit),
		Params: map[string]interface{}{"checkpoint": checkpoint},
	})
	defer it.Stop()

	actions := make([]*SwapAction, 0)
	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return actions, err
		}
		var action SwapAction
		err = row.ToStruct(&action)
		if err != nil {
			return actions, err
		}
		actions = append(actions, &action)
	}

	return actions, nil
}

func WriteSwapAction(ctx context.Context, a *SwapAction) error {
	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		exist, err := checkSwapActionExistence(ctx, txn, a.ActionId)
		if err != nil || exist {
			return err
		}
		a.CreatedAt = spanner.CommitTimestamp
		mutation, err := spanner.InsertStruct("swap_actions", a)
		if err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	return err
}

func checkSwapActionExistence(ctx context.Context, txn *spanner.ReadWriteTransaction, actionId string) (bool, error) {
	it := txn.Read(ctx, "swap_actions", spanner.Key{actionId}, []string{"created_at"})
	defer it.Stop()

	_, err := it.Next()
	if err == iterator.Done {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
