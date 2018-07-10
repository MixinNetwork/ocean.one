# Ocean ONE

Ocean ONE is a decentralized exchange built on Mixin Network, it's almost the first time that a decentralized exchange gain the same user experience as a centralized one.

Ocean ONE accepts all assets in Mixin Network as base currencies, and the only supported quote currencies are Bitcoin BTC (c6d0c728-2624-429b-8e0d-d9d19b6592fa) and Omni USDT (815b0b1a-2764-3736-8faa-42d694fa620a).

All order and trade data are encoded in the Mixin snapshots' memo field, the memo is base64 encoded [MessagePack](https://github.com/msgpack).


## Create Order

To sell 0.7 XIN with price 0.1 BTC/XIN, send a 0.7 XIN transfer to Ocean ONE with base64 encoded MessagePack data as the memo.

```golang
type OrderAction struct {
  S string    // side
  A uuid.UUID // asset
  P string    // price
  T string    // type
  O uuid.UUID // order
}

memo = base64.StdEncoding.EncodeToString(msgpack(OrderAction{
  T: "L",
  P: "0.1",
  S: "A",
  A: uuid.FromString("c6d0c728-2624-429b-8e0d-d9d19b6592fa"),
}))
```

To buy some XIN with price 0.1 BTC/XIN, send the desired amount of BTC transfer to Ocean ONE with base64 encoded MessagePack data as the memo.

```golang
memo = base64.StdEncoding.EncodeToString(msgpack(OrderAction{
  T: "L",
  P: "0.1",
  S: "B",
  A: uuid.FromString("c94ac88f-4671-3976-b60a-09064f1811e8"),
}))
```

It's recommended to set the `trace_id` field whenever you send a transfer to Ocean ONE, the `trace_id` will be used as the order id.


## Cancel Order

Send any amount of any asset to Ocean ONE with base64 encoded MessagePack data as the memo.

```golang
memo = base64.StdEncoding.EncodeToString(msgpack(OrderAction{
  O: uuid.FromString("2497b2bb-4d67-49bf-b2bc-211b0543d7ac"),
}))
```


## Bid Order Behavior

A bid order, despite a limit bid order or market bid order, will transfer some quote funds to the matching engine. Ocean ONE engine will match all the funds, this is a typical behavior for market order. However for a limit bid order, user may expect the order done whenever the desired bid size filled, in this situation, Ocean ONE engine still matches all the funds which may result in a larger order size filled.


## Events

The order book and all matches are always available in the Mixin Network snapshots, and Ocean ONE offers a WebSocket layer to provide a convenient query interface.

The WebSocket endipoint is `wss://events.ocean.one`, and all messages sent and received should be gziped. The event message is in a standard format.

```json
{
  "id": "a3fb2c7d-88ed-4605-977c-ebbb3f32ad71",
  "action": "EMIT_EVENT",
  "params": {},
  "data": {},
  "error": "description"
}
```

The `params` field is for the client sent message. The `data` or `error` is for the server message, and only one of them will be present in the message. If the message is the server response of a message from the client, the `id` and `action` fields will be identical to the sent one.

Whenever a client connects to the events server, it must send a `SUBSCRIBE_BOOK` message to the server, otherwise the client won't receive any events messages.

```json
{
  "id": "a3fb2c7d-88ed-4605-977c-ebbb3f32ad71",
  "action": "SUBSCRIBE_BOOK",
  "params": {
    "market": "c94ac88f-4671-3976-b60a-09064f1811e8-c6d0c728-2624-429b-8e0d-d9d19b6592fa"
  }
}
```

This will subscibe the client to all the events of the specific `market` in the `params`. To unsubscribe, send a similar message but with the action `UNSUBSCRIBE_BOOK`. A client can always subscribe to many markets with many different `SUBSCRIBE_BOOK` messages.


#### BOOK-T0

This is the first event whenever a client subscribe to a specific market, the event contains the full order book of the market.

```json
{
  "id": "a3fb2c7d-88ed-4605-977c-ebbb3f32ad71",
  "action": "EMIT_EVENT",
  "data": {
    "market": "c94ac88f-4671-3976-b60a-09064f1811e8-c6d0c728-2624-429b-8e0d-d9d19b6592fa",
    "sequence": 1531142594,
    "event": "BOOK-T0",
    "data": {
      "asks": [],
      "bids": []
    }
  }
}
```


#### ORDER-OPEN

The order is now open on the order book. This message will only be sent for orders which are not fully filled immediately. `amount` will indicate how much of the order is unfilled and going on the book.


#### ORDER-MATCH

A trade occurred between two orders. The taker order is the one executing immediately after being received and the maker order is a resting order on the book. The `side` field indicates the maker order side. If the side is `ask` this indicates the maker was a sell order and the match is considered an up-tick. A `bid` side match is a down-tick.


#### ORDER-CANCEL

The order is cancelled and no longer on the order book, `amount` indicates how much of the order went unfilled.


## Fee

- Taker: 0.1%
- Maker: 0.0%


## References

- Coinbase Pro API https://docs.pro.coinbase.com/
