package engine

import "github.com/MixinMessenger/go-number"

const (
	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"
)

type Order struct {
	Id              string
	Side            string
	Type            string
	Price           uint64
	FilledPrice     uint64
	RemainingAmount number.Decimal
	FilledAmount    number.Decimal

	Quote  string
	Base   string
	UserId string
}
