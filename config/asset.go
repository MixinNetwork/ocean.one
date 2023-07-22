package config

import (
	"log"

	"github.com/MixinNetwork/go-number"
)

const (
	MixinAssetId     = "c94ac88f-4671-3976-b60a-09064f1811e8"
	BitcoinAssetId   = "c6d0c728-2624-429b-8e0d-d9d19b6592fa"
	PUSDAssetId      = "31d2ea9c-95eb-3355-b65b-ba096853bc18"
	ERC20USDTAssetId = "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
)

func VerifyQuoteBase(quote, base string) bool {
	if quote == base {
		return false
	}
	if quote != BitcoinAssetId && quote != PUSDAssetId && quote != ERC20USDTAssetId && quote != MixinAssetId {
		return false
	}
	if quote == MixinAssetId && base == BitcoinAssetId {
		return false
	}
	if quote == MixinAssetId && base == PUSDAssetId {
		return false
	}
	if quote == MixinAssetId && base == ERC20USDTAssetId {
		return false
	}
	if quote == BitcoinAssetId && base == PUSDAssetId {
		return false
	}
	if quote == BitcoinAssetId && base == ERC20USDTAssetId {
		return false
	}
	if quote == PUSDAssetId && base == ERC20USDTAssetId {
		return false
	}
	return true
}

func QuotePrecision(assetId string) uint8 {
	switch assetId {
	case MixinAssetId:
		return 8
	case BitcoinAssetId:
		return 8
	case PUSDAssetId:
		return 4
	case ERC20USDTAssetId:
		return 4
	default:
		log.Panicln("QuotePrecision", assetId)
	}
	return 0
}

func QuoteMinimum(assetId string) number.Decimal {
	switch assetId {
	case MixinAssetId:
		return number.FromString("0.00000001")
	case BitcoinAssetId:
		return number.FromString("0.00000001")
	case PUSDAssetId:
		return number.FromString("0.0001")
	case ERC20USDTAssetId:
		return number.FromString("0.0001")
	default:
		log.Panicln("QuoteMinimum", assetId)
	}
	return number.Zero()
}
