package engine

const (
	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"
)

type Order struct {
	Id              string
	Side            string
	Type            string
	Price           uint64
	RemainingAmount uint64
	FilledAmount    uint64
	CreatedAt       uint64

	UserId string
}
