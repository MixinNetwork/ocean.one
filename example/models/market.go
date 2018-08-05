package models

type Market struct {
	Quote string
	Base  string
}

func AllMarkets() []*Market {
	var markets []*Market
	for _, b := range usdtMarkets {
		markets = append(markets, &Market{Quote: "815b0b1a-2764-3736-8faa-42d694fa620a", Base: b})
	}
	for _, b := range btcMarkets {
		markets = append(markets, &Market{Quote: "c6d0c728-2624-429b-8e0d-d9d19b6592fa", Base: b})
	}
	for _, b := range xinMarkets {
		markets = append(markets, &Market{Quote: "c94ac88f-4671-3976-b60a-09064f1811e8", Base: b})
	}
	return markets
}

var usdtMarkets = []string{
	"c6d0c728-2624-429b-8e0d-d9d19b6592fa", // BTC
	"fd11b6e3-0b87-41f1-a41f-f0e9b49e5bf0", // BCH
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d", // EOS
	"43d61dcd-e413-450d-80b8-101d5e903357", // ETH
	"2204c1ee-0ea2-4add-bb9a-b3719cfff93a", // ETC
	"76c802a2-7c88-447f-a93e-c29c9e5dd9c8", // LTC
	"23dfb5a5-5d7b-48b6-905f-3970e3176e27", // XRP
	"990c4c29-57e9-48f6-9819-7d986ea44985", // SC
	"c94ac88f-4671-3976-b60a-09064f1811e8", // XIN
}

var btcMarkets = []string{
	"fd11b6e3-0b87-41f1-a41f-f0e9b49e5bf0",
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d",
	"43d61dcd-e413-450d-80b8-101d5e903357",
	"2204c1ee-0ea2-4add-bb9a-b3719cfff93a",
	"76c802a2-7c88-447f-a93e-c29c9e5dd9c8",
	"23dfb5a5-5d7b-48b6-905f-3970e3176e27",
	"990c4c29-57e9-48f6-9819-7d986ea44985",
	"c94ac88f-4671-3976-b60a-09064f1811e8",
}

var xinMarkets = []string{
	"fd11b6e3-0b87-41f1-a41f-f0e9b49e5bf0",
	"6cfe566e-4aad-470b-8c9a-2fd35b49c68d",
	"43d61dcd-e413-450d-80b8-101d5e903357",
	"2204c1ee-0ea2-4add-bb9a-b3719cfff93a",
	"76c802a2-7c88-447f-a93e-c29c9e5dd9c8",
	"23dfb5a5-5d7b-48b6-905f-3970e3176e27",
	"990c4c29-57e9-48f6-9819-7d986ea44985",
}
