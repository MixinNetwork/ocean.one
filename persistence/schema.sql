CREATE TABLE actions (
  order_id     STRING(36) NOT NULL,
  action       STRING(36) NOT NULL,
  created_at   TIMESTAMP NOT NULL,
) PRIMARY KEY(order_id, action);

CREATE INDEX actions_by_created ON actions(created_at);


CREATE TABLE orders (
  order_id          STRING(36) NOT NULL,
  order_type        STRING(36) NOT NULL,
  side              STRING(36) NOT NULL,
  price             STRING(128) NOT NULL,
  remaining_amount  STRING(128) NOT NULL,
  filled_amount     STRING(128) NOT NULL,
  created_at        TIMESTAMP NOT NULL,
  user_id           STRING(36) NOT NULL,
) PRIMARY KEY(order_id);


CREATE TABLE trades (
  trade_id          STRING(36) NOT NULL,
  user_id           STRING(36) NOT NULL,
  ask_order_id      STRING(36) NOT NULL,
  bid_order_id      STRING(36) NOT NULL,
  ask_asset_id      STRING(36) NOT NULL,
  bid_asset_id      STRING(36) NOT NULL,
  side              STRING(36) NOT NULL,
  liquidity         STRING(36) NOT NULL,
  state             STRING(36) NOT NULL,
  price             STRING(128) NOT NULL,
  amount            STRING(128) NOT NULL,
  created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(trade_id, user_id);

CREATE INDEX trades_by_state_created ON trades(state, created_at);
