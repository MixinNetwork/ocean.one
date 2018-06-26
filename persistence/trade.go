package persistence

import (
	"time"
)

const (
	MakerFeeRate = "0.00000"
	TakerFeeRate = "0.00001"

	TradeLiquidityTaker = "TAKER"
	TradeLiquidityMaker = "MAKER"

	TransferSourceTrade = "TRADE"
	TransferSourceOrder = "ORDER"
)

type Trade struct {
	TradeId      string    `spanner:"trade_id"`
	Liquidity    string    `spanner:"liquidity"`
	AskOrderId   string    `spanner:"ask_order_id"`
	BidOrderId   string    `spanner:"bid_order_id"`
	QuoteAssetId string    `spanner:"quote_asset_id"`
	BaseAssetId  string    `spanner:"base_asset_id"`
	Side         string    `spanner:"side"`
	Price        string    `spanner:"price"`
	Amount       string    `spanner:"amount"`
	CreatedAt    time.Time `spanner:"created_at"`
	UserId       string    `spanner:"user_id"`
	FeeAssetId   string    `spanner:"fee_asset_id"`
	FeeAmount    string    `spanner:"fee_amount"`
}

type Transfer struct {
	TransferId string    `spanner:"transfer_id"`
	Source     string    `spanner:"source"`
	Detail     string    `spanner:"detail"`
	AssetId    string    `spanner:"asset_id"`
	Amount     string    `spanner:"amount"`
	CreatedAt  time.Time `spanner:"created_at"`
	UserId     string    `spanner:"user_id"`
}
