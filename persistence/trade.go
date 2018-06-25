package persistence

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/engine"
	"github.com/satori/go.uuid"
)

const (
	TradeLiquidityTaker = "TAKER"
	TradeLiquidityMaker = "MAKER"

	TradeStatePending = "PENDING"
	TradeStateDone    = "DONE"
)

type Trade struct {
	TradeId      string    `spanner:"trade_id"`
	Liquidity    string    `spanner:"liquidity"`
	AskOrderId   string    `spanner:"ask_order_id"`
	BidOrderId   string    `spanner:"bid_order_id"`
	QuoteAssetId string    `spanner:"quote_asset_id"`
	BaseAssetId  string    `spanner:"base_asset_id"`
	Side         string    `spanner:"side"`
	Price        string    `spanner:"price"`
	Amount       string    `spanner:"amount"`
	CreatedAt    time.Time `spanner:"created_at"`
	State        string    `spanner:"state"`
	UserId       string    `spanner:"user_id"`
}

func Transact(ctx context.Context, taker, maker *engine.Order, amount number.Decimal, precision int32) error {
	makerPrice := number.FromString(fmt.Sprint(maker.Price)).Mul(number.New(1, -precision)).Persist()
	makerFilledPrice := number.FromString(fmt.Sprint(maker.FilledPrice)).Mul(number.New(1, -precision)).Persist()
	takerFilledPrice := number.FromString(fmt.Sprint(taker.FilledPrice)).Mul(number.New(1, -precision)).Persist()

	takerOrderCols := []string{"order_id", "filled_amount", "remaining_amount", "filled_price"}
	takerOrderVals := []interface{}{taker.Id, taker.FilledAmount.Persist(), taker.RemainingAmount.Persist(), takerFilledPrice}
	makerOrderCols := []string{"order_id", "filled_amount", "remaining_amount", "filled_price"}
	makerOrderVals := []interface{}{maker.Id, maker.FilledAmount.Persist(), maker.RemainingAmount.Persist(), makerFilledPrice}
	if taker.RemainingAmount.Sign() == 0 {
		takerOrderCols = append(takerOrderCols, "state")
		takerOrderVals = append(takerOrderVals, OrderStateDone)
	}
	if maker.RemainingAmount.Sign() == 0 {
		makerOrderCols = append(makerOrderCols, "state")
		makerOrderVals = append(makerOrderVals, OrderStateDone)
	}

	tradeId, err := uuid.NewV4()
	if err != nil {
		return err
	}
	askOrderId, bidOrderId := taker.Id, maker.Id
	if taker.Side == engine.PageSideBid {
		askOrderId, bidOrderId = maker.Id, taker.Id
	}
	takerTrade := &Trade{
		TradeId:      tradeId.String(),
		Liquidity:    TradeLiquidityTaker,
		AskOrderId:   askOrderId,
		BidOrderId:   bidOrderId,
		QuoteAssetId: taker.Quote,
		BaseAssetId:  taker.Base,
		Side:         taker.Side,
		Price:        makerPrice,
		Amount:       amount.Persist(),
		CreatedAt:    time.Now(),
		State:        TradeStatePending,
		UserId:       taker.UserId,
	}
	makerTrade := &Trade{
		TradeId:      tradeId.String(),
		Liquidity:    TradeLiquidityMaker,
		AskOrderId:   askOrderId,
		BidOrderId:   bidOrderId,
		QuoteAssetId: maker.Quote,
		BaseAssetId:  maker.Base,
		Side:         maker.Side,
		Price:        makerPrice,
		Amount:       amount.Persist(),
		CreatedAt:    time.Now(),
		State:        TradeStatePending,
		UserId:       maker.UserId,
	}
	_, err = Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.Update("orders", takerOrderCols, takerOrderVals),
			spanner.Update("orders", makerOrderCols, makerOrderVals),
		}
		if taker.RemainingAmount.Sign() == 0 {
			mutations = append(mutations, spanner.Delete("actions", spanner.Key{taker.Id, engine.OrderActionCreate}))
			mutations = append(mutations, spanner.Delete("actions", spanner.Key{taker.Id, engine.OrderActionCancel}))
		}
		if maker.RemainingAmount.Sign() == 0 {
			mutations = append(mutations, spanner.Delete("actions", spanner.Key{maker.Id, engine.OrderActionCreate}))
			mutations = append(mutations, spanner.Delete("actions", spanner.Key{maker.Id, engine.OrderActionCancel}))
		}
		takerTradeMutation, err := spanner.InsertStruct("trades", takerTrade)
		if err != nil {
			return err
		}
		mutations = append(mutations, takerTradeMutation)
		makerTradeMutation, err := spanner.InsertStruct("trades", makerTrade)
		if err != nil {
			return err
		}
		mutations = append(mutations, makerTradeMutation)
		return txn.BufferWrite(mutations)
	})
	return err
}

func CancelOrder(ctx context.Context, order *engine.Order, precision int) error {
	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return nil
	})
	return err
}
