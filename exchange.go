package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MixinMessenger/bot-api-go-client"
	"github.com/shopspring/decimal"
	"github.com/ugorji/go/codec"
	"ocean.one/config"
	"ocean.one/engine"
	"ocean.one/persistence"
)

const (
	BitcoinAssetId = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	USDTAssetId    = "815b0b1a-2764-3736-8faa-42d694fa620a"
)

type Snapshot struct {
	SnapshotId string `json:"snapshot_id"`
	Amount     string `json:"amount"`
	Asset      struct {
		AssetId string `json:"asset_id"`
	} `json:"asset"`
	CreatedAt time.Time `json:"created_at"`

	TraceId    string `json:"trace_id"`
	UserId     string `json:"user_id"`
	OpponentId string `json:"opponent_id"`
	Data       string `json:"data"`
}

type OrderAction struct {
	Side    string
	AssetId string
	Price   string
	OrderId string
}

type Exchange struct {
	books map[string]*engine.Book
	codec codec.Handle
}

func NewExchange() *Exchange {
	return &Exchange{
		codec: new(codec.MsgpackHandle),
		books: make(map[string]*engine.Book),
	}
}

func (ex *Exchange) PollMixinNetwork(ctx context.Context) {
	checkpoint, limit := persistence.ReadLatestAction(ctx).UTC(), 500
	for {
		snapshots, err := ex.requestMixinNetwork(ctx, checkpoint, limit)
		if err != nil {
			log.Println("PollMixinNetwork ERROR", err)
			time.Sleep(1 * time.Second)
			continue
		}
		for _, s := range snapshots {
			log.Println(s)
			checkpoint = s.CreatedAt
			action := ex.validateSnapshot(ctx, s)
			if action == nil {
				ex.refund(ctx, s.OpponentId, s.Asset.AssetId, s.Amount)
				continue
			}
			if action.OrderId != "" {
				persistence.CancelOrder(ctx, action.OrderId)
				continue
			}
			amount := decimal.RequireFromString(s.Amount)
			price, _ := decimal.NewFromString(action.Price)
			if !price.IsPositive() {
				ex.refund(ctx, s.OpponentId, s.Asset.AssetId, s.Amount)
				continue
			}
			var quote, base string
			if action.Side == engine.PageSideAsk {
				quote, base = action.AssetId, s.Asset.AssetId
			} else if action.Side == engine.PageSideBid {
				quote, base = s.Asset.AssetId, action.AssetId
				amount = amount.Div(price)
			} else {
				ex.refund(ctx, s.OpponentId, s.Asset.AssetId, s.Amount)
				continue
			}
			if !ex.validateQuoteBasePair(quote, base) {
				ex.refund(ctx, s.OpponentId, s.Asset.AssetId, s.Amount)
				continue
			}
			persistence.CreateOrder(ctx, s.OpponentId, s.TraceId, action.Side, quote, base, amount, price, s.CreatedAt)
		}
		if len(snapshots) < limit {
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (ex *Exchange) validateQuoteBasePair(quote, base string) bool {
	if quote != BitcoinAssetId && quote != USDTAssetId {
		return false
	}
	if quote == BitcoinAssetId && base == USDTAssetId {
		return false
	}
	return true
}

func (ex *Exchange) validateSnapshot(ctx context.Context, s *Snapshot) *OrderAction {
	if s.UserId != config.ClientId {
		return nil
	}
	if decimal.RequireFromString(s.Amount).IsNegative() {
		return nil
	}
	if s.OpponentId == "" || s.TraceId == "" {
		return nil
	}
	action := ex.decryptOrderAction(ctx, s.Data)
	if action == nil {
		return nil
	}
	if action.AssetId == s.Asset.AssetId {
		return nil
	}
	return action
}

func (ex *Exchange) refund(ctx context.Context, opponentId, assetId, amount string) {
}

func (ex *Exchange) decryptOrderAction(ctx context.Context, data string) *OrderAction {
	payload, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil
	}
	var action OrderAction
	decoder := codec.NewDecoderBytes(payload, ex.codec)
	err = decoder.Decode(&action)
	if err != nil {
		return nil
	}
	return &action
}

func (ex *Exchange) requestMixinNetwork(ctx context.Context, checkpoint time.Time, limit int) ([]*Snapshot, error) {
	uri := fmt.Sprintf("/network/snapshots?offset=%s&order=ASC&limit=%d", checkpoint.Format(time.RFC3339Nano), limit)
	token, err := bot.SignAuthenticationToken(config.ClientId, config.SessionId, config.SessionKey, "GET", uri, "")
	if err != nil {
		return nil, err
	}
	body, err := bot.Request(ctx, "GET", uri, nil, token)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data  []*Snapshot `json:"data"`
		Error string      `json:"error"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return result.Data, nil
}
