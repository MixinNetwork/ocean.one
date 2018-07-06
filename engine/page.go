package engine

import (
	"log"

	"github.com/MixinMessenger/go-number"
	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/emirpasic/gods/trees/redblacktree"
)

const (
	PageSideAsk = "ASK"
	PageSideBid = "BID"
)

type Entry struct {
	Side   string         `json:"side"`
	Price  uint64         `json:"price"`
	Amount number.Decimal `json:"amount"`
	list   *arraylist.List
	orders map[string]*Order
}

type Page struct {
	Side    string
	points  *redblacktree.Tree
	entries map[uint64]*Entry
}

func NewPage(side string) *Page {
	if side != PageSideBid && side != PageSideAsk {
		return nil
	}
	return &Page{
		Side:    side,
		points:  redblacktree.NewWith(entryCompare),
		entries: make(map[uint64]*Entry),
	}
}

func (page *Page) Put(order *Order) {
	if page.Side != order.Side {
		log.Panicln(page, order)
	}
	entry, found := page.entries[order.Price]
	if !found {
		entry = &Entry{
			Side:   order.Side,
			Price:  order.Price,
			Amount: number.Zero(),
			list:   arraylist.New(),
			orders: make(map[string]*Order),
		}
		page.entries[entry.Price] = entry
		page.points.Put(entry, true)
	}
	if entry.Price != order.Price || entry.Side != order.Side {
		log.Panicln(entry, order)
	}
	if _, found := entry.orders[order.Id]; found {
		log.Panicln(order)
	}
	entry.Amount = entry.Amount.Add(order.RemainingAmount)
	entry.orders[order.Id] = order
	entry.list.Add(order.Id)
}

func (page *Page) Remove(order *Order) {
	if page.Side != order.Side {
		return
	}
	entry, found := page.entries[order.Price]
	if !found {
		return
	}
	if _, found := entry.orders[order.Id]; !found {
		return
	}
	index := entry.list.IndexOf(order.Id)
	if index < 0 {
		log.Panicln(order)
	}
	delete(entry.orders, order.Id)
	entry.Amount = entry.Amount.Sub(order.RemainingAmount)
	entry.list.Remove(index)
}

func (page *Page) Iterate(hook func(*Order) (number.Decimal, bool)) {
	for it := page.points.Iterator(); it.Next(); {
		entry := it.Key().(*Entry)
		for eit := entry.list.Iterator(); eit.Next(); {
			order := entry.orders[eit.Value().(string)]
			matchedAmount, done := hook(order)
			entry.Amount = entry.Amount.Sub(matchedAmount)
			if done {
				eit.End()
				it.End()
			}
		}
	}
}

func (page *Page) List(count int) []*Entry {
	entries := make([]*Entry, 0)
	for it := page.points.Iterator(); it.Next(); {
		entry := it.Key().(*Entry)
		entries = append(entries, &Entry{
			Side:   entry.Side,
			Price:  entry.Price,
			Amount: entry.Amount,
		})
		if count = count - 1; count == 0 {
			it.End()
		}
	}
	return entries
}

func entryCompare(a, b interface{}) int {
	entry := a.(*Entry)
	opponent := b.(*Entry)
	if entry.Price == opponent.Price {
		log.Panicln(entry, opponent)
	}
	switch entry.Side {
	case PageSideAsk:
		if entry.Price < opponent.Price {
			return -1
		}
	case PageSideBid:
		if entry.Price > opponent.Price {
			return -1
		}
	default:
		log.Panicln(entry, opponent)
	}
	return 1
}
