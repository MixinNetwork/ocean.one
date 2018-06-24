package engine

import (
	"log"

	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/emirpasic/gods/trees/redblacktree"
)

const (
	PageSideAsk = "ASK"
	PageSideBid = "BID"
)

type Entry struct {
	Side   string
	Price  uint64
	list   *arraylist.List
	orders map[string]*Order
}

type Page struct {
	Side    string
	points  *redblacktree.Tree
	entries map[uint64]*Entry
}

func NewPage(side string) *Page {
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
	entry.orders[order.Id] = order
	entry.list.Add(order.Id)
}

func (page *Page) Remove(order *Order) {
	if page.Side != order.Side {
		log.Panicln(page, order)
	}
	entry, found := page.entries[order.Price]
	if !found {
		log.Panicln(page, order)
	}
	if _, found := entry.orders[order.Id]; !found {
		log.Panicln(order)
	}
	if id, found := entry.list.Get(0); !found {
		log.Panicln(order)
	} else if id.(string) != order.Id {
		log.Panicln(order, id)
	}
	delete(entry.orders, order.Id)
	entry.list.Remove(0)
}

func (page *Page) Iterate(hook func(*Order) bool) {
	for it := page.points.Iterator(); it.Next(); {
		entry := it.Key().(*Entry)
		for eit := entry.list.Iterator(); eit.Next(); {
			order := entry.orders[eit.Value().(string)]
			if done := hook(order); done {
				eit.End()
				it.End()
			}
		}
	}
}

func entryCompare(a, b interface{}) int {
	entry := a.(*Entry)
	opponent := b.(*Entry)
	if entry.Side != opponent.Side {
		log.Panicln(entry, opponent)
	}
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
