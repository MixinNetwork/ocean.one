package persistence

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/engine"
	"github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

const (
	MakerFeeRate = "0.00000"
	TakerFeeRate = "0.00001"

	TradeLiquidityTaker = "TAKER"
	TradeLiquidityMaker = "MAKER"

	TransferSourceTradeConfirmed = "TRADE_CONFIRMED"
	TransferSourceOrderCancelled = "ORDER_CANCELLED"
	TransferSourceOrderFilled    = "ORDER_FILLED"
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

type Transfer struct {
	TransferId string    `spanner:"transfer_id"`
	Source     string    `spanner:"source"`
	Detail     string    `spanner:"detail"`
	AssetId    string    `spanner:"asset_id"`
	Amount     string    `spanner:"amount"`
	CreatedAt  time.Time `spanner:"created_at"`
	UserId     string    `spanner:"user_id"`
}

func ListPendingTransfers(ctx context.Context, limit int) ([]*Transfer, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	it := txn.Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf("SELECT transfer_id FROM transfers@{FORCE_INDEX=transfers_by_created} ORDER BY created_at LIMIT %d", limit),
	})
	defer it.Stop()

	transferIds := make([]string, 0)
	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		var id string
		err = row.Columns(&id)
		if err != nil {
			return nil, err
		}
		transferIds = append(transferIds, id)
	}

	tit := txn.Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM transfers WHERE transfer_id IN UNNEST(@transfer_ids)",
		Params: map[string]interface{}{"transfer_ids": transferIds},
	})
	defer tit.Stop()

	transfers := make([]*Transfer, 0)
	for {
		row, err := tit.Next()
		if err == iterator.Done {
			return transfers, nil
		} else if err != nil {
			return transfers, err
		}
		var transfer Transfer
		err = row.ToStruct(&transfer)
		if err != nil {
			return transfers, err
		}
		transfers = append(transfers, &transfer)
	}
}

func ExpireTransfers(ctx context.Context, transfers []*Transfer) error {
	var set []spanner.KeySet
	for _, t := range transfers {
		set = append(set, spanner.Key{t.TransferId})
	}
	_, err := Spanner(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Delete("transfers", spanner.KeySets(set...)),
	})
	return err
}

func Transact(ctx context.Context, taker, maker *engine.Order, amount number.Decimal, precision int32) error {
	askTrade, bidTrade := makeTrades(taker, maker, amount, precision)
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

	mutations := makeOrderMutations(taker, maker, precision)
	mutations = append(mutations, askTradeMutation, bidTradeMutation)
	mutations = append(mutations, askTransferMutation, bidTransferMutation)
	_, err = Spanner(ctx).Apply(ctx, mutations)
	return err
}

func CancelOrder(ctx context.Context, order *engine.Order, precision int32) error {
	price := number.FromString(fmt.Sprint(order.Price)).Mul(number.New(1, precision))
	filledPrice := number.FromString(fmt.Sprint(order.FilledPrice)).Mul(number.New(1, precision))
	orderCols := []string{"order_id", "filled_amount", "remaining_amount", "filled_price", "state"}
	orderVals := []interface{}{order.Id, order.FilledAmount.Persist(), order.RemainingAmount.Persist(), filledPrice.Persist(), OrderStateDone}
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
		transfer.Amount = price.Mul(order.RemainingAmount.Add(order.FilledAmount)).Sub(filledPrice.Mul(order.FilledAmount)).Persist()
	}
	transferMutation, err := spanner.InsertStruct("transfers", transfer)
	if err != nil {
		return err
	}
	mutations = append(mutations, transferMutation)
	_, err = Spanner(ctx).Apply(ctx, mutations)
	return err
}

func ReadTransferTrade(ctx context.Context, tradeId, assetId string) (*Trade, error) {
	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM trades WHERE trade_id=@trade_id",
		Params: map[string]interface{}{"trade_id": tradeId},
	})
	defer it.Stop()

	for {
		row, err := it.Next()
		if err == iterator.Done {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		var trade Trade
		err = row.ToStruct(&trade)
		if err != nil {
			return nil, err
		}
		if trade.FeeAssetId == assetId {
			return &trade, nil
		}
	}
}

func makeOrderMutations(taker, maker *engine.Order, precision int32) []*spanner.Mutation {
	makerFilledPrice := number.FromString(fmt.Sprint(maker.FilledPrice)).Mul(number.New(1, precision)).Persist()
	takerFilledPrice := number.FromString(fmt.Sprint(taker.FilledPrice)).Mul(number.New(1, precision)).Persist()

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
	mutations := []*spanner.Mutation{
		spanner.Update("orders", takerOrderCols, takerOrderVals),
		spanner.Update("orders", makerOrderCols, makerOrderVals),
	}

	if taker.RemainingAmount.Sign() == 0 {
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{taker.Id, engine.OrderActionCreate}))
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{taker.Id, engine.OrderActionCancel}))
	}
	if taker.Side == engine.PageSideBid && taker.RemainingAmount.Sign() == 0 && taker.Price > taker.FilledPrice {
		change := number.FromString(fmt.Sprint(taker.Price - taker.FilledPrice)).Mul(number.New(1, precision))
		transfer := &Transfer{
			TransferId: getSettlementId(taker.Id, engine.OrderActionCancel),
			Source:     TransferSourceOrderFilled,
			Detail:     taker.Id,
			AssetId:    taker.Quote,
			Amount:     change.Mul(taker.FilledAmount).Persist(),
			CreatedAt:  time.Now(),
			UserId:     taker.UserId,
		}
		transferMutation, err := spanner.InsertStruct("transfers", transfer)
		if err != nil {
			log.Panicln(err)
		}
		mutations = append(mutations, transferMutation)
	}
	if maker.RemainingAmount.Sign() == 0 {
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{maker.Id, engine.OrderActionCreate}))
		mutations = append(mutations, spanner.Delete("actions", spanner.Key{maker.Id, engine.OrderActionCancel}))
	}
	return mutations
}

func makeTrades(taker, maker *engine.Order, amount number.Decimal, precision int32) (*Trade, *Trade) {
	tradeId, _ := uuid.NewV4()
	askOrderId, bidOrderId := taker.Id, maker.Id
	if taker.Side == engine.PageSideBid {
		askOrderId, bidOrderId = maker.Id, taker.Id
	}
	price := number.FromString(fmt.Sprint(maker.Price)).Mul(number.New(1, precision))

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
