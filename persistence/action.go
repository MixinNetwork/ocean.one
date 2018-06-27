package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/engine"
	"google.golang.org/api/iterator"
)

const (
	OrderStatePending = "PENDING"
	OrderStateDone    = "DONE"
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
	State           string    `spanner:"state"`
	UserId          string    `spanner:"user_id"`
}

type Action struct {
	OrderId   string    `spanner:"order_id"`
	Action    string    `spanner:"action"`
	CreatedAt time.Time `spanner:"created_at"`

	Order *Order `spanner:"-"`
}

func ReadActionCheckpoint(ctx context.Context) (time.Time, error) {
	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{
		SQL: "SELECT created_at FROM actions ORDER BY created_at DESC LIMIT 1",
	})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return time.Now(), nil
	} else if err != nil {
		return time.Time{}, err
	}
	var checkpoint time.Time
	err = row.Columns(&checkpoint)
	return checkpoint, err
}

func ListPendingActions(ctx context.Context, checkpoint time.Time, limit int) ([]*Action, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	it := txn.Query(ctx, spanner.Statement{
		SQL:    fmt.Sprintf("SELECT * FROM actions@{FORCE_INDEX=actions_by_created} WHERE created_at>=@checkpoint ORDER BY created_at LIMIT %d", limit),
		Params: map[string]interface{}{"checkpoint": checkpoint},
	})
	defer it.Stop()

	orderFilters := make(map[string]bool)
	actions, orderIds := make([]*Action, 0), make([]string, 0)
	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return actions, err
		}
		var action Action
		err = row.ToStruct(&action)
		if err != nil {
			return actions, err
		}
		actions = append(actions, &action)
		if orderFilters[action.OrderId] {
			continue
		}
		orderFilters[action.OrderId] = true
		orderIds = append(orderIds, action.OrderId)
	}

	oit := txn.Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM orders WHERE order_id IN UNNEST(@order_ids)",
		Params: map[string]interface{}{"order_ids": orderIds},
	})
	defer oit.Stop()

	orders := make(map[string]*Order)
	for {
		row, err := oit.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return actions, err
		}
		var order Order
		err = row.ToStruct(&order)
		if err != nil {
			return actions, err
		}
		orders[order.OrderId] = &order
	}

	for _, a := range actions {
		a.Order = orders[a.OrderId]
	}
	return actions, nil
}

func CreateOrderAction(ctx context.Context, userId, traceId string, orderType, side, quote, base string, amount, price number.Decimal, createdAt time.Time) error {
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
		State:           OrderStatePending,
		UserId:          userId,
	}
	action := Action{
		OrderId:   order.OrderId,
		Action:    engine.OrderActionCreate,
		CreatedAt: createdAt,
	}
	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		state, err := checkOrderState(ctx, txn, action.OrderId)
		if err != nil || state != "" {
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

func CancelOrderAction(ctx context.Context, orderId string, createdAt time.Time) error {
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
		state, err := checkOrderState(ctx, txn, action.OrderId)
		if err != nil || state != OrderStatePending {
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

func checkOrderState(ctx context.Context, txn *spanner.ReadWriteTransaction, orderId string) (string, error) {
	it := txn.Read(ctx, "orders", spanner.Key{orderId}, []string{"state"})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return "", nil
	} else if err != nil {
		return "", err
	}
	var state string
	err = row.Columns(&state)
	return state, err
}
