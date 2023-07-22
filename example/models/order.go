package models

import (
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/engine"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/uuid"
	"github.com/ugorji/go/codec"
)

type OrderAction struct {
	TraceId string `json:"trace_id"`
	Quote   string `json:"quote"`
	Base    string `json:"base"`
	Side    string `json:"side"`
	Price   string `json:"price"`
	Amount  string `json:"amount"`
	Funds   string `json:"funds"`
	Type    string `json:"type"`
}

func (current *User) CreateOrder(ctx context.Context, o *OrderAction) error {
	if id, err := uuid.FromString(o.TraceId); err != nil {
		return session.BadDataError(ctx)
	} else {
		o.TraceId = id.String()
	}
	if id, err := uuid.FromString(o.Quote); err != nil {
		return session.BadDataError(ctx)
	} else {
		o.Quote = id.String()
	}
	if id, err := uuid.FromString(o.Base); err != nil {
		return session.BadDataError(ctx)
	} else {
		o.Base = id.String()
	}
	if o.Quote == config.OOOAssetId || o.Base == config.OOOAssetId {
		return session.ForbiddenError(ctx)
	}
	if !config.VerifyQuoteBase(o.Quote, o.Base) {
		return session.ForbiddenError(ctx)
	}

	precision := int32(config.QuotePrecision(o.Quote))
	price := number.FromString(o.Price).RoundFloor(precision)
	switch o.Type {
	case engine.OrderTypeLimit:
		o.Type = "L"
		if price.Exhausted() {
			return session.BadDataError(ctx)
		}
	case engine.OrderTypeMarket:
		o.Type = "M"
		if !price.IsZero() {
			return session.BadDataError(ctx)
		}
	default:
		return session.BadDataError(ctx)
	}

	amount := number.FromString(o.Amount).RoundFloor(4)
	sent, get := o.Quote, o.Base
	switch o.Side {
	case engine.PageSideBid:
		o.Side = "B"
		funds := number.FromString(o.Funds).RoundFloor(8)
		if price.IsPositive() && funds.Div(price).RoundFloor(4).Exhausted() {
			return session.BadDataError(ctx)
		}
		amount = funds
	case engine.PageSideAsk:
		o.Side = "A"
		sent, get = o.Base, o.Quote
	default:
		return session.BadDataError(ctx)
	}
	if amount.Exhausted() {
		return session.BadDataError(ctx)
	}

	action := map[string]interface{}{
		"S": o.Side,
		"P": price.Persist(),
		"T": o.Type,
	}
	action["A"], _ = uuid.FromString(get)
	memo := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&memo, handle)
	err := encoder.Encode(action)
	if err != nil {
		return session.ServerError(ctx, err)
	}

	return current.Key.sendTransfer(ctx, config.RandomBrokerId(), sent, amount, o.TraceId, base64.StdEncoding.EncodeToString(memo))
}

func (current *User) CancelOrder(ctx context.Context, id string) error {
	oid, _ := uuid.FromString(id)
	if oid.String() == uuid.Nil.String() {
		return session.BadDataError(ctx)
	}
	action := map[string]interface{}{"O": oid}
	memo := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&memo, handle)
	err := encoder.Encode(action)
	if err != nil {
		return session.ServerError(ctx, err)
	}

	return current.Key.sendTransfer(ctx, config.RandomBrokerId(), config.OOOAssetId, number.FromString("0.00000001"), uuid.NewV4().String(), base64.StdEncoding.EncodeToString(memo))
}
