package persistence

import (
	"context"
	"time"

	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/engine"
)

type Persist interface {
	ReadProperty(ctx context.Context, key string) (string, error)
	WriteProperty(ctx context.Context, key, value string) error
	ReadPropertyAsTime(ctx context.Context, key string) (time.Time, error)
	WriteTimeProperty(ctx context.Context, key string, value time.Time) error

	ListPendingActions(ctx context.Context, checkpoint time.Time, limit int) ([]*Action, error)
	CreateOrderAction(ctx context.Context, userId, traceId string, orderType, side, quote, base string, amount, price number.Decimal, createdAt time.Time) error
	CancelOrderAction(ctx context.Context, orderId string, createdAt time.Time) error

	ListPendingTransfers(ctx context.Context, limit int) ([]*Transfer, error)
	ExpireTransfers(ctx context.Context, transfers []*Transfer) error
	Transact(ctx context.Context, taker, maker *engine.Order, amount number.Decimal, precision int32) error
	CancelOrder(ctx context.Context, order *engine.Order, precision int32) error
	ReadTransferTrade(ctx context.Context, tradeId, assetId string) (*Trade, error)
}
