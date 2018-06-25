package persistence

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/engine"
	"google.golang.org/api/iterator"
)

type Order struct {
	OrderId         string    `spanner:"order_id"`
	OrderType       string    `spanner:"order_type"`
	QuoteAssetId    string    `spanner:"quote_asset_id"`
	BaseAssetId     string    `spanner:"base_asset_id"`
	Side            string    `spanner:"side"`
	Price           string    `spanner:"price"`
	FilledPrice     string    `spanner:"filled_price"`
	RemainingAmount string    `spanner:"remaining_amount"`
	FilledAmount    string    `spanner:"filled_amount"`
	CreatedAt       time.Time `spanner:"created_at"`
	UserId          string    `spanner:"user_id"`
}

type Action struct {
	OrderId   string    `spanner:"order_id"`
	Action    string    `spanner:"action"`
	CreatedAt time.Time `spanner:"created_at"`
}

func ReadActionCheckpoint(ctx context.Context) time.Time {
	return time.Now()
}

func CreateOrder(ctx context.Context, userId, traceId string, orderType, side, quote, base string, amount, price number.Decimal, createdAt time.Time) error {
	order := Order{
		OrderId:         traceId,
		OrderType:       orderType,
		Side:            side,
		QuoteAssetId:    quote,
		BaseAssetId:     base,
		Price:           price.Persist(),
		RemainingAmount: amount.Persist(),
		FilledAmount:    number.Zero().Persist(),
		CreatedAt:       createdAt,
		UserId:          userId,
	}
	action := Action{
		OrderId:   order.OrderId,
		Action:    engine.OrderActionCreate,
		CreatedAt: createdAt,
	}
	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		exist, err := checkAction(ctx, txn, action.OrderId, action.Action)
		if err != nil || exist {
			return err
		}
		orderMutation, err := spanner.InsertStruct("orders", order)
		if err != nil {
			return err
		}
		actionMutation, err := spanner.InsertStruct("actions", action)
		if err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{orderMutation, actionMutation})
	})
	return err
}

func CancelOrder(ctx context.Context, orderId string, createdAt time.Time) error {
	action := Action{
		OrderId:   orderId,
		Action:    engine.OrderActionCancel,
		CreatedAt: createdAt,
	}
	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		exist, err := checkAction(ctx, txn, action.OrderId, action.Action)
		if err != nil || exist {
			return err
		}
		exist, err = checkOrder(ctx, txn, action.OrderId)
		if err != nil || !exist {
			return err
		}
		actionMutation, err := spanner.InsertStruct("actions", action)
		if err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{actionMutation})
	})
	return err
}

func checkAction(ctx context.Context, txn *spanner.ReadWriteTransaction, orderId, action string) (bool, error) {
	it := txn.Read(ctx, "actions", spanner.Key{orderId, action}, []string{"created_at"})
	defer it.Stop()

	_, err := it.Next()
	if err == iterator.Done {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func checkOrder(ctx context.Context, txn *spanner.ReadWriteTransaction, orderId string) (bool, error) {
	it := txn.Read(ctx, "orders", spanner.Key{orderId}, []string{"created_at"})
	defer it.Stop()

	_, err := it.Next()
	if err == iterator.Done {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
