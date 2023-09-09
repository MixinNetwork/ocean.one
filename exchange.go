package main

import (
	"context"
	"encoding/base64"
	"log"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/config"
	"github.com/MixinNetwork/ocean.one/engine"
	"github.com/MixinNetwork/ocean.one/persistence"
	"github.com/gofrs/uuid/v5"
	"github.com/ugorji/go/codec"
)

const (
	PollInterval                    = 100 * time.Millisecond
	CheckpointMixinNetworkSnapshots = "exchange-checkpoint-mixin-network-snapshots"
)

type Exchange struct {
	books     map[string]*engine.Book
	codec     codec.Handle
	snapshots map[string]bool
	brokers   map[string]*persistence.Broker
	mutexes   *tmap
}

func NewExchange() *Exchange {
	return &Exchange{
		codec:     new(codec.MsgpackHandle),
		books:     make(map[string]*engine.Book),
		snapshots: make(map[string]bool),
		brokers:   make(map[string]*persistence.Broker),
		mutexes:   newTmap(),
	}
}

func (ex *Exchange) Run(ctx context.Context) {
	brokers, err := persistence.AllBrokers(ctx, true)
	if err != nil {
		log.Panicln(err)
	}
	for _, b := range brokers {
		ex.brokers[b.BrokerId] = b
		go ex.PollTransfers(ctx, b.BrokerId)
	}
	go ex.PollMixinMessages(ctx)
	go ex.PollMixinNetwork(ctx)
	ex.PollOrderActions(ctx)
}

func (ex *Exchange) PollOrderActions(ctx context.Context) {
	checkpoint, limit := time.Time{}, 500
	for {
		actions, err := persistence.ListPendingActions(ctx, checkpoint, limit)
		if err != nil {
			log.Println("ListPendingActions", err)
			time.Sleep(PollInterval)
			continue
		}
		for _, a := range actions {
			ex.ensureProcessOrderAction(ctx, a)
			checkpoint = a.CreatedAt
		}
		if len(actions) < limit {
			time.Sleep(PollInterval)
		}
	}
}

func (ex *Exchange) PollTransfers(ctx context.Context, brokerId string) {
	limit := 500
	for {
		transfers, err := persistence.ListPendingTransfers(ctx, brokerId, limit)
		if err != nil {
			log.Println("ListPendingTransfers", brokerId, err)
			time.Sleep(PollInterval)
			continue
		}
		for _, t := range transfers {
			ex.ensureProcessTransfer(ctx, t)
		}
		for {
			err = persistence.ExpireTransfers(ctx, transfers)
			if err == nil {
				break
			}
			log.Println("ExpireTransfers", err)
			time.Sleep(PollInterval)
		}
		if len(transfers) < limit {
			time.Sleep(PollInterval)
		}
	}
}

type TransferAction struct {
	S string    // source
	O uuid.UUID // cancelled order
	A uuid.UUID // matched ask order
	B uuid.UUID // matched bid order
	F string    // fee
}

func (ex *Exchange) ensureProcessTransfer(ctx context.Context, transfer *persistence.Transfer) {
	if number.FromString(transfer.Amount).Exhausted() {
		log.Println("processTransfer skipped", "TransferId:", transfer.TransferId)
		return
	}

	for {
		err := ex.processTransfer(ctx, transfer)
		if err == nil {
			break
		}
		log.Println("processTransfer", err, "TransferId:", transfer.TransferId)
		time.Sleep(PollInterval)
	}
}

func (ex *Exchange) processTransfer(ctx context.Context, transfer *persistence.Transfer) error {
	var data *TransferAction
	switch transfer.Source {
	case persistence.TransferSourceOrderFilled:
		data = &TransferAction{S: "FILL", O: uuid.FromStringOrNil(transfer.Detail)}
	case persistence.TransferSourceOrderCancelled:
		data = &TransferAction{S: "CANCEL", O: uuid.FromStringOrNil(transfer.Detail)}
	case persistence.TransferSourceOrderInvalid:
		data = &TransferAction{S: "REFUND", O: uuid.FromStringOrNil(transfer.Detail), F: transfer.Fee}
	case persistence.TransferSourceTradeConfirmed:
		trade, err := persistence.ReadTransferTrade(ctx, transfer.Detail, transfer.AssetId)
		if err != nil {
			return err
		}
		if trade == nil {
			log.Panicln(transfer)
		}
		data = &TransferAction{S: "MATCH", A: uuid.FromStringOrNil(trade.AskOrderId), B: uuid.FromStringOrNil(trade.BidOrderId), F: transfer.Fee}
	default:
		log.Panicln(transfer)
	}
	out := make([]byte, 140)
	encoder := codec.NewEncoderBytes(&out, ex.codec)
	err := encoder.Encode(data)
	if err != nil {
		log.Panicln(err)
	}
	memo := base64.StdEncoding.EncodeToString(out)
	if len(memo) > 140 {
		log.Panicln(transfer, memo)
	}
	return ex.sendTransfer(ctx, transfer.BrokerId, transfer.UserId, transfer.AssetId, number.FromString(transfer.Amount), transfer.TransferId, memo)
}

func (ex *Exchange) buildBook(ctx context.Context, market string) *engine.Book {
	return engine.NewBook(ctx, market, func(taker, maker *engine.Order, amount number.Integer) string {
		for {
			tradeId, err := persistence.Transact(ctx, taker, maker, amount)
			if err == nil {
				return tradeId
			}
			log.Println("Engine Transact CALLBACK", err)
			time.Sleep(PollInterval)
		}
	}, func(order *engine.Order) {
		for {
			err := persistence.CancelOrder(ctx, order)
			if err == nil {
				break
			}
			log.Println("Engine Cancel CALLBACK", err)
			time.Sleep(PollInterval)
		}
	})
}

func (ex *Exchange) ensureProcessOrderAction(ctx context.Context, action *persistence.Action) {
	order := action.Order
	market := order.BaseAssetId + "-" + order.QuoteAssetId
	book := ex.books[market]
	if book == nil {
		book = ex.buildBook(ctx, market)
		go book.Run(ctx)
		ex.books[market] = book
	}
	pricePrecision := config.QuotePrecision(order.QuoteAssetId)
	fundsPrecision := pricePrecision + AmountPrecision
	price := number.FromString(order.Price).Integer(pricePrecision)
	remainingAmount := number.FromString(order.RemainingAmount).Integer(AmountPrecision)
	filledAmount := number.FromString(order.FilledAmount).Integer(AmountPrecision)
	remainingFunds := number.FromString(order.RemainingFunds).Integer(fundsPrecision)
	filledFunds := number.FromString(order.FilledFunds).Integer(fundsPrecision)
	book.AttachOrderEvent(ctx, &engine.Order{
		Id:              order.OrderId,
		Side:            order.Side,
		Type:            order.OrderType,
		Price:           price,
		RemainingAmount: remainingAmount,
		FilledAmount:    filledAmount,
		RemainingFunds:  remainingFunds,
		FilledFunds:     filledFunds,
		Quote:           order.QuoteAssetId,
		Base:            order.BaseAssetId,
		UserId:          order.UserId,
		BrokerId:        order.BrokerId,
	}, action.Action)
}

func (ex *Exchange) PollMixinNetwork(ctx context.Context) {
	const limit = 500
	for {
		checkpoint, err := persistence.ReadPropertyAsTime(ctx, CheckpointMixinNetworkSnapshots)
		if err != nil {
			log.Println("ReadPropertyAsTime CheckpointMixinNetworkSnapshots", err)
			time.Sleep(PollInterval)
			continue
		}
		if checkpoint.IsZero() {
			checkpoint = time.Now().UTC()
		}
		snapshots, err := ex.requestMixinNetwork(ctx, checkpoint, limit)
		if err != nil {
			log.Println("PollMixinNetwork ERROR", err)
			time.Sleep(PollInterval)
			continue
		}
		for _, s := range snapshots {
			if ex.snapshots[s.SnapshotId] {
				continue
			}
			ex.ensureProcessSnapshot(ctx, s)
			checkpoint = s.CreatedAt
			ex.snapshots[s.SnapshotId] = true
		}
		if len(snapshots) < limit {
			time.Sleep(PollInterval)
		}
		err = persistence.WriteTimeProperty(ctx, CheckpointMixinNetworkSnapshots, checkpoint)
		if err != nil {
			log.Println("WriteTimeProperty CheckpointMixinNetworkSnapshots", err)
		}
	}
}

func (ex *Exchange) PollMixinMessages(ctx context.Context) {
	for {
		blazeClient := bot.NewBlazeClient(config.ClientId, config.SessionId, config.SessionKey)
		err := blazeClient.Loop(ctx, ex)
		if err != nil {
			log.Println("PollMixinMessages", err)
			time.Sleep(1 * time.Second)
		}
	}
}

func (ex *Exchange) OnMessage(ctx context.Context, msg bot.MessageView, userId string) error {
	return nil
}

func (ex *Exchange) OnAckReceipt(ctx context.Context, msg bot.MessageView, userId string) error {
	return nil
}

func (ex *Exchange) SyncAck() bool {
	return false
}
