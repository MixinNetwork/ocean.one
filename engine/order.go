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
	Price           number.Integer
	FilledPrice     number.Integer
	RemainingAmount number.Integer
	FilledAmount    number.Integer
	RemainingFunds  number.Integer
	FilledFunds     number.Integer

	Quote  string
	Base   string
	UserId string
}
