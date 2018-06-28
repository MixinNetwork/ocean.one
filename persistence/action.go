package persistence

import (
	"time"
)

const (
	OrderStatePending = "PENDING"
	OrderStateDone    = "DONE"
)

type Order struct {
	OrderId         string    `spanner:"order_id"`
	OrderType       string    `spanner:"order_type"`
	QuoteAssetId    string    `spanner:"quote_asset_id"`
	BaseAssetId     string    `spanner:"base_asset_id"`
	Side            string    `spanner:"side"`
	Price           string    `spanner:"price"`
	FilledPrice     string    `spanner:"filled_price"`
	RemainingAmount string    `spanner:"remaining_amount"`
	FilledAmount    string    `spanner:"filled_amount"`
	CreatedAt       time.Time `spanner:"created_at"`
	State           string    `spanner:"state"`
	UserId          string    `spanner:"user_id"`
}

type Action struct {
	OrderId   string    `spanner:"order_id"`
	Action    string    `spanner:"action"`
	CreatedAt time.Time `spanner:"created_at"`

	Order *Order `spanner:"-"`
}
