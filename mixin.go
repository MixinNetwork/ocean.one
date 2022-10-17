package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/config"
	"github.com/MixinNetwork/ocean.one/engine"
	"github.com/MixinNetwork/ocean.one/persistence"
	"github.com/gofrs/uuid"
	"github.com/ugorji/go/codec"
)

const (
	AmountPrecision = 8
	RefundRate      = "0.999"
	MaxPrice        = 1000000000
	MaxAmount       = 50000000000000

	MixinAssetId     = "c94ac88f-4671-3976-b60a-09064f1811e8"
	BitcoinAssetId   = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	USDTAssetId      = "815b0b1a-2764-3736-8faa-42d694fa620a"
	PUSDAssetId      = "31d2ea9c-95eb-3355-b65b-ba096853bc18"
	ERC20USDTAssetId = "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
)

type Error struct {
	Status      int    `json:"status"`
	Code        int    `json:"code"`
	Description string `json:"description"`
}

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
	U []byte    // user
	S string    // side
	A uuid.UUID // asset
	P string    // price
	T string    // type
	O uuid.UUID // order
}

func (ex *Exchange) ensureProcessSnapshot(ctx context.Context, s *Snapshot) {
	for {
		err := ex.processSnapshot(ctx, s)
		if err == nil {
			break
		}
		log.Println("ensureProcessSnapshot", err)
		time.Sleep(100 * time.Millisecond)
	}
}

func (ex *Exchange) processSnapshot(ctx context.Context, s *Snapshot) error {
	if ex.brokers[s.UserId] == nil {
		return nil
	}
	if s.OpponentId == "" || s.TraceId == "" {
		return nil
	}
	if number.FromString(s.Amount).Exhausted() {
		return nil
	}

	action, err := ex.decryptOrderAction(ctx, s.Data)
	if err != nil {
		return ex.refundSnapshot(ctx, s)
	}
	if len(action.U) > 16 {
		return persistence.UpdateUserPublicKey(ctx, s.OpponentId, hex.EncodeToString(action.U))
	}
	if action.O.String() != uuid.Nil.String() {
		return persistence.CancelOrderAction(ctx, action.O.String(), s.CreatedAt, s.OpponentId)
	}

	if action.A.String() == s.Asset.AssetId {
		return ex.refundSnapshot(ctx, s)
	}
	if action.T != engine.OrderTypeLimit && action.T != engine.OrderTypeMarket {
		return ex.refundSnapshot(ctx, s)
	}

	quote, base := ex.getQuoteBasePair(s, action)
	if quote == "" {
		return ex.refundSnapshot(ctx, s)
	}

	priceDecimal := number.FromString(action.P)
	maxPrice := number.NewDecimal(MaxPrice, int32(QuotePrecision(quote)))
	if priceDecimal.Cmp(maxPrice) > 0 {
		return ex.refundSnapshot(ctx, s)
	}
	price := priceDecimal.Integer(QuotePrecision(quote))
	if action.T == engine.OrderTypeLimit {
		if price.IsZero() {
			return ex.refundSnapshot(ctx, s)
		}
	} else if !price.IsZero() {
		return ex.refundSnapshot(ctx, s)
	}

	fundsPrecision := AmountPrecision + QuotePrecision(quote)
	funds := number.NewInteger(0, fundsPrecision)
	amount := number.NewInteger(0, AmountPrecision)

	assetDecimal := number.FromString(s.Amount)
	if action.S == engine.PageSideBid {
		maxAmount := number.NewDecimal(MaxAmount, AmountPrecision)
		maxFunds := maxPrice.Mul(maxAmount)
		if assetDecimal.Cmp(maxFunds) > 0 {
			return ex.refundSnapshot(ctx, s)
		}
		funds = assetDecimal.Integer(fundsPrecision)
		if funds.Decimal().Cmp(QuoteMinimum(quote)) < 0 {
			return ex.refundSnapshot(ctx, s)
		}
	} else {
		maxAmount := number.NewDecimal(MaxAmount, AmountPrecision)
		if assetDecimal.Cmp(maxAmount) > 0 {
			return ex.refundSnapshot(ctx, s)
		}
		amount = assetDecimal.Integer(AmountPrecision)
		if action.T == engine.OrderTypeLimit && price.Mul(amount).Decimal().Cmp(QuoteMinimum(quote)) < 0 {
			return ex.refundSnapshot(ctx, s)
		}
	}

	return persistence.CreateOrderAction(ctx, &engine.Order{
		Id:              s.TraceId,
		Type:            action.T,
		Side:            action.S,
		Quote:           quote,
		Base:            base,
		Price:           price,
		RemainingAmount: amount,
		FilledAmount:    amount.Zero(),
		RemainingFunds:  funds,
		FilledFunds:     funds.Zero(),
	}, s.OpponentId, s.UserId, s.CreatedAt)
}

func (ex *Exchange) getQuoteBasePair(s *Snapshot, a *OrderAction) (string, string) {
	var quote, base string
	if a.S == engine.PageSideAsk {
		quote, base = a.A.String(), s.Asset.AssetId
	} else if a.S == engine.PageSideBid {
		quote, base = s.Asset.AssetId, a.A.String()
	} else {
		return "", ""
	}
	if quote == base {
		return "", ""
	}
	if quote != BitcoinAssetId && quote != USDTAssetId && quote != PUSDAssetId && quote != ERC20USDTAssetId && quote != MixinAssetId {
		return "", ""
	}
	if quote == BitcoinAssetId && base == USDTAssetId {
		return "", ""
	}
	if quote == BitcoinAssetId && base == PUSDAssetId {
		return "", ""
	}
	if quote == BitcoinAssetId && base == ERC20USDTAssetId {
		return "", ""
	}
	if quote == MixinAssetId && base == USDTAssetId {
		return "", ""
	}
	if quote == MixinAssetId && base == PUSDAssetId {
		return "", ""
	}
	if quote == MixinAssetId && base == ERC20USDTAssetId {
		return "", ""
	}
	if quote == MixinAssetId && base == BitcoinAssetId {
		return "", ""
	}
	if quote == PUSDAssetId && base == USDTAssetId {
		return "", ""
	}
	if quote == PUSDAssetId && base == ERC20USDTAssetId {
		return "", ""
	}
	if quote == ERC20USDTAssetId && base == USDTAssetId {
		return "", ""
	}
	return quote, base
}

func (ex *Exchange) refundSnapshot(ctx context.Context, s *Snapshot) error {
	amount := number.FromString(s.Amount).Mul(number.FromString(RefundRate))
	if amount.Exhausted() {
		return nil
	}
	fee := number.FromString(s.Amount).Sub(amount)
	return persistence.CreateRefundTransfer(ctx, s.UserId, s.OpponentId, s.Asset.AssetId, amount, fee, s.TraceId)
}

func (ex *Exchange) decryptOrderAction(ctx context.Context, data string) (*OrderAction, error) {
	payload, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(data)
		if err != nil {
			return nil, err
		}
	}
	var action OrderAction
	decoder := codec.NewDecoderBytes(payload, ex.codec)
	err = decoder.Decode(&action)
	if err != nil {
		return nil, err
	}
	switch action.T {
	case "L":
		action.T = engine.OrderTypeLimit
	case "M":
		action.T = engine.OrderTypeMarket
	}
	switch action.S {
	case "A":
		action.S = engine.PageSideAsk
	case "B":
		action.S = engine.PageSideBid
	}
	return &action, nil
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
	var resp struct {
		Data  []*Snapshot `json:"data"`
		Error string      `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Data, nil
}

func (ex *Exchange) sendTransfer(ctx context.Context, brokerId, recipientId, assetId string, amount number.Decimal, traceId, memo string) error {
	mutex := ex.mutexes.fetch(recipientId, assetId)
	mutex.Lock()
	defer mutex.Unlock()

	broker := ex.brokers[brokerId]
	_, err := bot.CreateTransfer(ctx, &bot.TransferInput{
		AssetId:     assetId,
		RecipientId: recipientId,
		Amount:      amount,
		TraceId:     traceId,
		Memo:        memo,
	}, broker.BrokerId, broker.SessionId, broker.SessionKey, broker.DecryptedPIN, broker.PINToken)
	return err
}

type tmap struct {
	sync.Map
}

func newTmap() *tmap {
	return &tmap{
		Map: sync.Map{},
	}
}

func (m *tmap) fetch(user, asset string) *sync.Mutex {
	uu, err := uuid.FromString(user)
	if err != nil {
		panic(user)
	}
	u := new(big.Int).SetBytes(uu.Bytes())
	au, err := uuid.FromString(asset)
	if err != nil {
		panic(asset)
	}
	a := new(big.Int).SetBytes(au.Bytes())
	s := new(big.Int).Add(u, a)
	key := new(big.Int).Mod(s, big.NewInt(100)).String()
	if _, found := m.Load(key); !found {
		m.Store(key, new(sync.Mutex))
	}
	val, _ := m.Load(key)
	return val.(*sync.Mutex)
}
