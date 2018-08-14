Example Front End
=================

A basic front end to provide users and developers a familiar exchange interface. This front end has some selected markets and can be queried with a simple /markets API.

```
GET https://example.ocean.one/markets

{
  "data": [
    {
      "base": "43d61dcd-e413-450d-80b8-101d5e903357",
      "base_symbol": "ETH",
      "change": -0.002649317907466239,
      "price": 317.4274,
      "quote": "815b0b1a-2764-3736-8faa-42d694fa620a",
      "quote_symbol": "USDT",
      "quote_usd": 1.0174087,
      "total": 4163484.3716705,
      "volume": 12991.477100000004
    }
  ]
}
```

| field        | description                     |
|--------------|---------------------------------|
| base         | base currency ID                |
| base_symbol  | base currency symbol            |
| quote        | quote currency ID               |
| quote_symbol | quote currency symbol           |
| price        | latest price in quote currency  |
| change       | 24 hour price change            |
| volume       | 24 hour volume in base currency |
| total        | 24 hour total in quote currency |

To query a specific market data, combine the base and quote currency ID as the market ID

```
GET https://example.ocean.one/markets/MARKET-ID
```
