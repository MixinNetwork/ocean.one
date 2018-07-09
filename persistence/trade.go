package persistence

import (
	"context"
	"crypto/md5"
	"io"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/engine"
	"github.com/satori/go.uuid"
)

const (
	MakerFeeRate = "0.000"
	TakerFeeRate = "0.001"

	TradeLiquidityTaker = "TAKER"
	TradeLiquidityMaker = "MAKER"
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
	UserId       string    `spanner:"user_id"`
	FeeAssetId   string    `spanner:"fee_asset_id"`
	FeeAmount    string    `spanner:"fee_amount"`
}

func Transact(ctx context.Context, taker, maker *engine.Order, amount, funds number.Integer) error {
	askTrade, bidTrade := makeTrades(taker, maker, amount.Decimal())
	askTransfer, bidTransfer := handleFees(askTrade, bidTrade)

	askTradeMutation, err := spanner.InsertStruct("trades", askTrade)
	if err != nil {
		return err
	}
	bidTradeMutation, err := spanner.InsertStruct("trades", bidTrade)
	if err != nil {
		return err
	}

	askTransferMutation, err := spanner.InsertStruct("transfers", askTransfer)
	if err != nil {
		return err
	}
	bidTransferMutation, err := spanner.InsertStruct("transfers", bidTransfer)
	if err != nil {
		return err
	}

	mutations := makeOrderMutations(taker, maker)
	mutations = append(mutations, askTradeMutation, bidTradeMutation)
	mutations = append(mutations, askTransferMutation, bidTransferMutation)
	_, err = Spanner(ctx).Apply(ctx, mutations)
	return err
}

func CancelOrder(ctx context.Context, order *engine.Order) error {
	orderCols := []string{"order_id", "filled_amount", "remaining_amount", "filled_funds", "remaining_funds", "state"}
	orderVals := []interface{}{order.Id, order.FilledAmount.Persist(), order.RemainingAmount.Persist(), order.FilledFunds.Persist(), order.RemainingFunds.Persist(), OrderStateDone}
	mutations := []*spanner.Mutation{
		spanner.Update("orders", orderCols, orderVals),
		spanner.Delete("actions", spanner.Key{order.Id, engine.OrderActionCreate}),
		spanner.Delete("actions", spanner.Key{order.Id, engine.OrderActionCancel}),
	}

	transfer := &Transfer{
		TransferId: getSettlementId(order.Id, engine.OrderActionCancel),
		Source:     TransferSourceOrderCancelled,
		Detail:     order.Id,
		AssetId:    order.Base,
		Amount:     order.RemainingAmount.Persist(),
		CreatedAt:  time.Now(),
		UserId:     order.UserId,
	}
	if order.Side == engine.PageSideBid {
		transfer.AssetId = order.Quote
		transfer.Amount = order.RemainingFunds.Persist()
	}
	transferMutation, err := spanner.InsertStruct("transfers", transfer)
	if err != nil {
		return err
	}
	mutations = append(mutations, transferMutation)
	_, err = Spanner(ctx).Apply(ctx, mutations)
	return err
}

func makeOrderMutations(taker, maker *engine.Order) []*spanner.Mutation {
	takerOrderCols := []string{"order_id", "filled_amount", "remaining_amount", "filled_funds", "remaining_funds"}
	takerOrderVals := []interface{}{taker.Id, taker.FilledAmount.Persist(), taker.RemainingAmount.Persist(), taker.FilledFunds.Persist(), taker.RemainingFunds.Persist()}
	makerOrderCols := []string{"order_id", "filled_amount", "remaining_amount", "filled_funds", "remaining_funds"}
	makerOrderVals := []interface{}{maker.Id, maker.FilledAmount.Persist(), maker.RemainingAmount.Persist(), maker.FilledFunds.Persist(), maker.RemainingFunds.Persist()}
	if taker.RemainingAmount.IsZero() && taker.RemainingFunds.IsZero() {
		takerOrderCols = append(takerOrderCols, "state")
		takerOrderVals = append(takerOrderVals, OrderStateDone)
	}
	if maker.RemainingAmount.IsZero() && maker.RemainingFunds.IsZero() {
		makerOrderCols = append(makerOrderCols, "state")
		makerOrderVals = append(makerOrderVals, OrderStateDone)
	}
	mutations := []*spanner.Mutation{
		spanner.Update("orders", takerOrderCols, takerOrderVals),
		spanner.Update("orders", makerOrderCols, makerOrderVals),
	}

	if taker.RemainingAmount.IsZero() && taker.RemainingFunds.IsZero() {
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{taker.Id, engine.OrderActionCreate}))
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{taker.Id, engine.OrderActionCancel}))
	}
	if maker.RemainingAmount.IsZero() && maker.RemainingFunds.IsZero() {
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{maker.Id, engine.OrderActionCreate}))
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{maker.Id, engine.OrderActionCancel}))
	}
	return mutations
}

func makeTrades(taker, maker *engine.Order, amount number.Decimal) (*Trade, *Trade) {
	tradeId, _ := uuid.NewV4()
	askOrderId, bidOrderId := taker.Id, maker.Id
	if taker.Side == engine.PageSideBid {
		askOrderId, bidOrderId = maker.Id, taker.Id
	}
	price := maker.Price.Decimal()

	takerTrade := &Trade{
		TradeId:      tradeId.String(),
		Liquidity:    TradeLiquidityTaker,
		AskOrderId:   askOrderId,
		BidOrderId:   bidOrderId,
		QuoteAssetId: taker.Quote,
		BaseAssetId:  taker.Base,
		Side:         taker.Side,
		Price:        price.Persist(),
		Amount:       amount.Persist(),
		CreatedAt:    time.Now(),
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
		Price:        price.Persist(),
		Amount:       amount.Persist(),
		CreatedAt:    time.Now(),
		UserId:       maker.UserId,
	}

	askTrade, bidTrade := takerTrade, makerTrade
	if askTrade.Side == engine.PageSideBid {
		askTrade, bidTrade = makerTrade, takerTrade
	}
	return askTrade, bidTrade
}

func handleFees(ask, bid *Trade) (*Transfer, *Transfer) {
	total := number.FromString(ask.Amount).Mul(number.FromString(ask.Price))
	askFee := total.Mul(number.FromString(TakerFeeRate))
	bidFee := number.FromString(bid.Amount).Mul(number.FromString(MakerFeeRate))
	if ask.Liquidity == TradeLiquidityMaker {
		askFee = total.Mul(number.FromString(MakerFeeRate))
		bidFee = number.FromString(bid.Amount).Mul(number.FromString(TakerFeeRate))
	}

	ask.FeeAssetId = ask.QuoteAssetId
	ask.FeeAmount = askFee.Persist()
	bid.FeeAssetId = bid.BaseAssetId
	bid.FeeAmount = bidFee.Persist()

	askTransfer := &Transfer{
		TransferId: getSettlementId(ask.TradeId, ask.Liquidity),
		Source:     TransferSourceTradeConfirmed,
		Detail:     ask.TradeId,
		AssetId:    ask.FeeAssetId,
		Amount:     total.Sub(askFee).Persist(),
		CreatedAt:  time.Now(),
		UserId:     ask.UserId,
	}
	bidTransfer := &Transfer{
		TransferId: getSettlementId(bid.TradeId, bid.Liquidity),
		Source:     TransferSourceTradeConfirmed,
		Detail:     bid.TradeId,
		AssetId:    bid.FeeAssetId,
		Amount:     number.FromString(bid.Amount).Sub(bidFee).Persist(),
		CreatedAt:  time.Now(),
		UserId:     bid.UserId,
	}
	return askTransfer, bidTransfer
}

func getSettlementId(id, modifier string) string {
	h := md5.New()
	io.WriteString(h, id)
	io.WriteString(h, modifier)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}
