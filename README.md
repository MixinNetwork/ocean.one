# ocean.one

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


## Fee

- Taker: 0.01%
- Maker: 0%
