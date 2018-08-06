package models

import (
	"context"
	"strings"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/uuid"
)

func (current *User) CreateWithdrawal(ctx context.Context, assetId string, amount number.Decimal, traceId, memo string) error {
	if id, err := uuid.FromString(assetId); err != nil {
		return session.BadDataError(ctx)
	} else {
		assetId = id.String()
	}
	if id, err := uuid.FromString(traceId); err != nil {
		return session.BadDataError(ctx)
	} else {
		traceId = id.String()
	}
	if amount.Exhausted() {
		return session.BadDataError(ctx)
	}
	if len(memo) > 140 {
		return session.BadDataError(ctx)
	}
	if !current.MixinId.Valid {
		return session.MixinNotConnectedError(ctx)
	}
	userId := strings.Split(current.MixinId.StringVal, ":")[0]
	return current.Key.sendTransfer(ctx, userId, assetId, amount, traceId, memo)
}
