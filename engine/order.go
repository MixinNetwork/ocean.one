package engine

import (
	"log"

	"github.com/MixinNetwork/go-number"
)

const (
	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"
)

type Order struct {
	Id              string
	Side            string
	Type            string
	Price           number.Integer
	RemainingAmount number.Integer
	FilledAmount    number.Integer
	RemainingFunds  number.Integer
	FilledFunds     number.Integer

	Quote  string
	Base   string
	UserId string
}

func (order *Order) filled() bool {
	if order.Side == PageSideAsk {
		return order.RemainingAmount.IsZero()
	}
	return order.RemainingFunds.IsZero()
}

func (order *Order) assert() {
	switch order.Side {
	case PageSideAsk:
		if !order.RemainingFunds.IsZero() {
			log.Panicln(order)
		}
	case PageSideBid:
		if !order.RemainingAmount.IsZero() {
			log.Panicln(order)
		}
	default:
		log.Panicln(order)
	}

	switch order.Type {
	case OrderTypeLimit:
		if order.Price.IsZero() {
			log.Panicln(order)
		}
	case OrderTypeMarket:
		if !order.Price.IsZero() {
			log.Panicln(order)
		}
	default:
		log.Panicln(order)
	}
}
