package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MixinMessenger/go-number"
	"github.com/MixinMessenger/ocean.one/cache"
)

const (
	OrderActionCreate = "CREATE"
	OrderActionCancel = "CANCEL"

	EventQueueSize = 8192
)

type TransactCallback func(taker, maker *Order, amount number.Integer)
type CancelCallback func(order *Order)

type OrderEvent struct {
	Order  *Order
	Action string
}

type Book struct {
	market      string
	events      chan *OrderEvent
	createIndex map[string]bool
	cancelIndex map[string]bool
	transact    TransactCallback
	cancel      CancelCallback
	asks        *Page
	bids        *Page
	queue       *cache.Queue
}

func NewBook(ctx context.Context, market string, transact TransactCallback, cancel CancelCallback) *Book {
	return &Book{
		market:      market,
		events:      make(chan *OrderEvent, EventQueueSize),
		createIndex: make(map[string]bool),
		cancelIndex: make(map[string]bool),
		transact:    transact,
		cancel:      cancel,
		asks:        NewPage(PageSideAsk),
		bids:        NewPage(PageSideBid),
		queue:       cache.NewQueue(ctx, market),
	}
}

func (book *Book) AttachOrderEvent(ctx context.Context, order *Order, action string) {
	if action != OrderActionCreate && action != OrderActionCancel {
		log.Panicln(order, action)
	}
	if order.Side != PageSideAsk && order.Side != PageSideBid {
		log.Panicln(order, action)
	}
	if order.Type != OrderTypeLimit && order.Type != OrderTypeMarket {
		log.Panicln(order, action)
	}
	book.events <- &OrderEvent{Order: order, Action: action}
}

func (book *Book) process(ctx context.Context, taker, maker *Order) number.Integer {
	matchedAmount := taker.RemainingAmount
	if maker.RemainingAmount.Cmp(matchedAmount) < 0 {
		matchedAmount = maker.RemainingAmount
	}
	matchedPrice := maker.Price
	matchedFunds := matchedPrice.Mul(matchedAmount)

	taker.RemainingAmount = taker.RemainingAmount.Sub(matchedAmount)
	taker.FilledAmount = taker.FilledAmount.Add(matchedAmount)
	taker.FilledFunds = taker.FilledFunds.Add(matchedFunds)
	taker.FilledPrice = taker.FilledFunds.Div(taker.FilledAmount)

	maker.RemainingAmount = maker.RemainingAmount.Sub(matchedAmount)
	maker.FilledAmount = maker.FilledAmount.Add(matchedAmount)
	maker.FilledFunds = maker.FilledFunds.Add(matchedFunds)
	maker.FilledPrice = maker.FilledFunds.Div(maker.FilledAmount)

	book.transact(taker, maker, matchedAmount)
	return matchedAmount
}

func (book *Book) createOrder(ctx context.Context, order *Order) {
	if _, found := book.createIndex[order.Id]; found {
		return
	}
	book.createIndex[order.Id] = true

	if order.Side == PageSideAsk {
		opponents := make([]*Order, 0)
		book.bids.Iterate(func(opponent *Order) (number.Integer, bool) {
			if order.Type == OrderTypeLimit && opponent.Price.Cmp(order.Price) < 0 {
				return number.NewInteger(0, order.RemainingAmount.Precision()), true
			}
			matchedAmount := book.process(ctx, order, opponent)
			book.cacheOrderEvent(ctx, cache.EventTypeOrderMatch, opponent.Side, opponent.Price, matchedAmount)
			opponents = append(opponents, opponent)
			return matchedAmount, order.RemainingAmount.IsZero()
		})
		for _, o := range opponents {
			if o.RemainingAmount.IsZero() {
				book.bids.Remove(o)
			}
		}
		if order.RemainingAmount.IsPositive() {
			if order.Type == OrderTypeLimit {
				book.asks.Put(order)
				book.cacheOrderEvent(ctx, cache.EventTypeOrderOpen, order.Side, order.Price, order.RemainingAmount)
			} else {
				book.cancel(order)
			}
		}
	} else if order.Side == PageSideBid {
		opponents := make([]*Order, 0)
		book.asks.Iterate(func(opponent *Order) (number.Integer, bool) {
			if order.Type == OrderTypeLimit && opponent.Price.Cmp(order.Price) > 0 {
				return number.NewInteger(0, order.RemainingAmount.Precision()), true
			}
			matchedAmount := book.process(ctx, order, opponent)
			book.cacheOrderEvent(ctx, cache.EventTypeOrderMatch, opponent.Side, opponent.Price, matchedAmount)
			opponents = append(opponents, opponent)
			return matchedAmount, order.RemainingAmount.IsZero()
		})
		for _, o := range opponents {
			if o.RemainingAmount.IsZero() {
				book.asks.Remove(o)
			}
		}
		if order.RemainingAmount.IsPositive() {
			if order.Type == OrderTypeLimit {
				book.bids.Put(order)
				book.cacheOrderEvent(ctx, cache.EventTypeOrderOpen, order.Side, order.Price, order.RemainingAmount)
			} else {
				book.cancel(order)
			}
		}
	}
}

func (book *Book) cancelOrder(ctx context.Context, order *Order) {
	if _, found := book.cancelIndex[order.Id]; found {
		return
	}
	book.cancelIndex[order.Id] = true
	book.cancel(order)
	book.cacheOrderEvent(ctx, cache.EventTypeOrderCancel, order.Side, order.Price, order.RemainingAmount)

	if order.Side == PageSideAsk {
		book.asks.Remove(order)
	} else if order.Side == PageSideBid {
		book.bids.Remove(order)
	} else {
		log.Panicln(order)
	}
}

func (book *Book) Run(ctx context.Context) {
	go book.queue.Loop(ctx)

	fullCacheTicker := time.NewTicker(time.Second * 30)
	defer fullCacheTicker.Stop()

	for {
		select {
		case event := <-book.events:
			if event.Action == OrderActionCreate {
				book.createOrder(ctx, event.Order)
			} else if event.Action == OrderActionCancel {
				book.cancelOrder(ctx, event.Order)
			} else {
				log.Panicln(event)
			}
		case <-fullCacheTicker.C:
			book.cacheList(ctx, 0)
		}
	}
}

func (book *Book) cacheList(ctx context.Context, limit int) {
	event := fmt.Sprintf("BOOK-T%d", limit)
	data := map[string]interface{}{
		"asks": book.asks.List(limit),
		"bids": book.bids.List(limit),
	}
	book.queue.AttachEvent(ctx, event, data)
}

func (book *Book) cacheOrderEvent(ctx context.Context, event, side string, price, amount number.Integer) {
	data := map[string]interface{}{
		"side":   side,
		"price":  price,
		"amount": amount,
	}
	book.queue.AttachEvent(ctx, event, data)
}
