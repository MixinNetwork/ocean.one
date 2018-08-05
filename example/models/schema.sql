CREATE TABLE properties (
	key         STRING(512) NOT NULL,
	value       STRING(8192) NOT NULL,
	updated_at  TIMESTAMP NOT NULL,
) PRIMARY KEY(key);


CREATE TABLE pool_keys (
	user_id	          STRING(36) NOT NULL,
	session_id        STRING(36) NOT NULL,
	session_key       STRING(1024) NOT NULL,
	pin_token         STRING(512) NOT NULL,
	encrypted_pin     STRING(512) NOT NULL,
	encryption_header BYTES(1024) NOT NULL,
	ocean_key         STRING(1024) NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id);


CREATE TABLE verifications (
	verification_id   STRING(36) NOT NULL,
	category          STRING(36) NOT NULL,
	receiver          STRING(512) NOT NULL,
	code              STRING(128) NOT NULL,
	provider          STRING(128) NOT NULL,
	created_at        TIMESTAMP NOT NULL,
	verified_at       TIMESTAMP,
) PRIMARY KEY(verification_id);

CREATE INDEX verifications_by_receiver_created_desc ON verifications(receiver, created_at DESC);


CREATE TABLE users (
	user_id	           STRING(36) NOT NULL,
	email              STRING(512),
	phone              STRING(512),
	mixin_id           STRING(36),
	identity_id        STRING(36),
	full_name          STRING(512) NOT NULL,
	encrypted_password STRING(1024) NOT NULL,
	active_at          TIMESTAMP NOT NULL,
	created_at         TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id);

CREATE UNIQUE NULL_FILTERED INDEX users_by_email ON users(email);
CREATE UNIQUE NULL_FILTERED INDEX users_by_phone ON users(phone);
CREATE UNIQUE NULL_FILTERED INDEX users_by_mixin_id ON users(mixin_id);


CREATE TABLE keys (
	user_id	          STRING(36) NOT NULL,
	session_id        STRING(36) NOT NULL,
	session_key       STRING(1024) NOT NULL,
	pin_token         STRING(512) NOT NULL,
	encrypted_pin     STRING(512) NOT NULL,
	encryption_header BYTES(1024) NOT NULL,
	ocean_key         STRING(1024) NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id),
INTERLEAVE IN PARENT users ON DELETE CASCADE;


CREATE TABLE sessions (
	user_id	          STRING(36) NOT NULL,
	session_id        STRING(36) NOT NULL,
	secret            STRING(512) NOT NULL,
	remote_address    STRING(1024) NOT NULL,
	active_at         TIMESTAMP NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id, session_id),
INTERLEAVE IN PARENT users ON DELETE CASCADE;


CREATE TABLE candles (
  base        STRING(36) NOT NULL,
  quote       STRING(36) NOT NULL,
  granularity INT64 NOT NULL,
  point       INT64 NOT NULL,
  open        FLOAT64 NOT NULL,
  close       FLOAT64 NOT NULL,
  high        FLOAT64 NOT NULL,
  low         FLOAT64 NOT NULL,
  volume      FLOAT64 NOT NULL,
  total       FLOAT64 NOT NULL,
) PRIMARY KEY(base, quote, granularity, point);


CREATE TABLE markets (
  base        STRING(36) NOT NULL,
  quote       STRING(36) NOT NULL,
  price       FLOAT64 NOT NULL,
  volume      FLOAT64 NOT NULL,
  total       FLOAT64 NOT NULL,
  change      FLOAT64 NOT NULL,
) PRIMARY KEY(base, quote);
