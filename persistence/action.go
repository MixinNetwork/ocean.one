package persistence

import (
	"context"
	"time"

	"github.com/MixinMessenger/go-number"
)

func ReadActionCheckpoint(ctx context.Context) time.Time {
	return time.Now()
}

func CreateOrder(ctx context.Context, userId, traceId string, orderType, side, quote, base string, amount, price number.Decimal, createdAt time.Time) error {
	return nil
}

func CancelOrder(ctx context.Context, orderId string, createdAt time.Time) error {
	return nil
}
