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

const (
	MixinAssetId   = "c94ac88f-4671-3976-b60a-09064f1811e8"
	BitcoinAssetId = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	USDTAssetId    = "815b0b1a-2764-3736-8faa-42d694fa620a"
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
	if !validateQuoteBase(o.Quote, o.Base) {
		return session.ForbiddenError(ctx)
	}

	price := number.FromString(o.Price).RoundFloor(8)
	if o.Quote == USDTAssetId {
		price = price.RoundFloor(4)
	}
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

	amount := number.FromString(o.Amount)
	sent, get := o.Quote, o.Base
	switch o.Side {
	case engine.PageSideBid:
		o.Side = "B"
		amount = number.FromString(o.Funds)
	case engine.PageSideAsk:
		o.Side = "A"
		sent, get = o.Base, o.Quote
	default:
		return session.BadDataError(ctx)
	}
	amount = amount.RoundFloor(4)
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

func validateQuoteBase(quote, base string) bool {
	if quote == base {
		return false
	}
	if quote != BitcoinAssetId && quote != USDTAssetId && quote != MixinAssetId {
		return false
	}
	if quote == BitcoinAssetId && base == USDTAssetId {
		return false
	}
	if quote == MixinAssetId && base == USDTAssetId {
		return false
	}
	if quote == MixinAssetId && base == BitcoinAssetId {
		return false
	}
	return true
}
