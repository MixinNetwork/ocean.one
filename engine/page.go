package engine

import (
	"log"

	"github.com/MixinNetwork/go-number"
	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/emirpasic/gods/trees/redblacktree"
)

const (
	PageSideAsk = "ASK"
	PageSideBid = "BID"
)

type Entry struct {
	Side   string         `json:"side"`
	Price  number.Integer `json:"price"`
	Amount number.Decimal `json:"amount"`
	Funds  number.Decimal `json:"funds"`
	list   *arraylist.List
	orders map[string]*Order
}

type Page struct {
	Side    string
	points  *redblacktree.Tree
	entries map[int64]*Entry
}

func NewPage(side string) *Page {
	if side != PageSideBid && side != PageSideAsk {
		return nil
	}
	return &Page{
		Side:    side,
		points:  redblacktree.NewWith(entryCompare),
		entries: make(map[int64]*Entry),
	}
}

func (page *Page) Put(order *Order) {
	if page.Side != order.Side {
		log.Panicln(page, order)
	}
	entry, found := page.entries[order.Price.Value()]
	if !found {
		entry = &Entry{
			Side:   order.Side,
			Price:  order.Price,
			Amount: number.Zero(),
			Funds:  number.Zero(),
			list:   arraylist.New(),
			orders: make(map[string]*Order),
		}
		page.entries[entry.Price.Value()] = entry
		page.points.Put(entry, true)
	}
	if entry.Price.Cmp(order.Price) != 0 || entry.Side != order.Side {
		log.Panicln(entry, order)
	}
	if _, found := entry.orders[order.Id]; found {
		log.Panicln(order)
	}
	if entry.Side == PageSideAsk {
		entry.Amount = entry.Amount.Add(order.RemainingAmount.Decimal())
	} else {
		entry.Funds = entry.Funds.Add(order.RemainingFunds.Decimal())
	}
	entry.orders[order.Id] = order
	entry.list.Add(order.Id)
}

func (page *Page) Remove(o *Order) *Order {
	if page.Side != o.Side {
		return nil
	}
	entry, found := page.entries[o.Price.Value()]
	if !found {
		return nil
	}
	order, found := entry.orders[o.Id]
	if !found {
		return nil
	}
	index := entry.list.IndexOf(order.Id)
	if index < 0 {
		log.Panicln(order)
	}
	delete(entry.orders, order.Id)
	if entry.Side == PageSideAsk {
		entry.Amount = entry.Amount.Sub(order.RemainingAmount.Decimal())
	} else {
		entry.Funds = entry.Funds.Sub(order.RemainingFunds.Decimal())
	}
	entry.list.Remove(index)
	return order
}

func (page *Page) Iterate(hook func(*Order) (number.Integer, number.Integer, bool)) {
	for it := page.points.Iterator(); it.Next(); {
		entry := it.Key().(*Entry)
		for eit := entry.list.Iterator(); eit.Next(); {
			order := entry.orders[eit.Value().(string)]
			matchedAmount, matchedFunds, done := hook(order)
			if entry.Side == PageSideAsk {
				entry.Amount = entry.Amount.Sub(matchedAmount.Decimal())
			} else {
				entry.Funds = entry.Funds.Sub(matchedFunds.Decimal())
			}
			if done {
				eit.End()
				it.End()
			}
		}
	}
}

func (page *Page) List(count int, filterEmpty bool) []*Entry {
	entries := make([]*Entry, 0)
	for it := page.points.Iterator(); it.Next(); {
		ie := it.Key().(*Entry)
		entry := &Entry{
			Side:   ie.Side,
			Price:  ie.Price,
			Amount: ie.Amount,
			Funds:  ie.Funds,
		}
		price := ie.Price.Decimal()
		if entry.Amount.IsZero() {
			entry.Amount = entry.Funds.Div(price)
		} else if entry.Funds.IsZero() {
			entry.Funds = price.Mul(entry.Amount)
		}
		if !filterEmpty || !entry.Funds.IsZero() {
			entries = append(entries, entry)
		}
		if count = count - 1; count == 0 {
			it.End()
		}
	}
	return entries
}

func entryCompare(a, b interface{}) int {
	entry := a.(*Entry)
	opponent := b.(*Entry)
	if entry.Price.Cmp(opponent.Price) == 0 {
		log.Panicln(entry, opponent)
	}
	switch entry.Side {
	case PageSideAsk:
		return entry.Price.Cmp(opponent.Price)
	case PageSideBid:
		return opponent.Price.Cmp(entry.Price)
	default:
		log.Panicln(entry, opponent)
		return 0
	}
}
