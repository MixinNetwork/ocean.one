package config

import "github.com/MixinNetwork/ocean.one/config"

func VerifyQuoteBase(quote, base string) bool {
	return config.VerifyQuoteBase(quote, base)
}

func QuotePrecision(assetId string) uint8 {
	return config.QuotePrecision(assetId)
}
