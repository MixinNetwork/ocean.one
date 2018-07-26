CREATE TABLE properties (
	key         STRING(512) NOT NULL,
	value       STRING(8192) NOT NULL,
	updated_at  TIMESTAMP NOT NULL,
) PRIMARY KEY(key);


CREATE TABLE orders (
  order_id          STRING(36) NOT NULL,
  order_type        STRING(36) NOT NULL,
  quote_asset_id    STRING(36) NOT NULL,
  base_asset_id     STRING(36) NOT NULL,
  side              STRING(36) NOT NULL,
  price             STRING(128) NOT NULL,
  remaining_amount  STRING(128) NOT NULL,
  filled_amount     STRING(128) NOT NULL,
  remaining_funds   STRING(128) NOT NULL,
  filled_funds      STRING(128) NOT NULL,
  created_at        TIMESTAMP NOT NULL,
  state             STRING(36) NOT NULL,
  user_id           STRING(36) NOT NULL,
) PRIMARY KEY(order_id);

CREATE INDEX orders_by_user_created_desc ON orders(user_id, created_at DESC) STORING(quote_asset_id,base_asset_id,state);


CREATE TABLE actions (
  order_id     STRING(36) NOT NULL,
  action       STRING(36) NOT NULL,
  created_at   TIMESTAMP NOT NULL,
) PRIMARY KEY(order_id, action),
INTERLEAVE IN PARENT orders ON DELETE CASCADE;

CREATE INDEX actions_by_created ON actions(created_at);


CREATE TABLE trades (
  trade_id          STRING(36) NOT NULL,
  liquidity         STRING(36) NOT NULL,
  ask_order_id      STRING(36) NOT NULL,
  bid_order_id      STRING(36) NOT NULL,
  quote_asset_id    STRING(36) NOT NULL,
  base_asset_id     STRING(36) NOT NULL,
  side              STRING(36) NOT NULL,
  price             STRING(128) NOT NULL,
  amount            STRING(128) NOT NULL,
  created_at        TIMESTAMP NOT NULL,
  user_id           STRING(36) NOT NULL,
  fee_asset_id      STRING(36) NOT NULL,
  fee_amount        STRING(128) NOT NULL,
) PRIMARY KEY(trade_id, liquidity);

CREATE INDEX trades_by_base_quote_created_desc ON trades(base_asset_id, quote_asset_id, created_at DESC);


CREATE TABLE transfers (
  transfer_id       STRING(36) NOT NULL,
  source            STRING(36) NOT NULL,
  detail            STRING(36) NOT NULL,
  asset_id          STRING(36) NOT NULL,
  amount            STRING(128) NOT NULL,
  created_at        TIMESTAMP NOT NULL,
  user_id           STRING(36) NOT NULL,
) PRIMARY KEY(transfer_id);

CREATE INDEX transfers_by_created ON transfers(created_at);


CREATE TABLE users (
  user_id          STRING(36) NOT NULL,
  public_key       STRING(512) NOT NULL,
) PRIMARY KEY(user_id);

CREATE UNIQUE INDEX users_by_public_key ON users(public_key);
