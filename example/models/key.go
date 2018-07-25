package models

const keys_DDL = `
CREATE TABLE keys (
	user_id	          STRING(36) NOT NULL,
	session_id        STRING(36) NOT NULL,
	private_key       STRING(1024) NOT NULL,
	pin_token         STRING(512) NOT NULL,
	encrypted_pin     STRING(512) NOT NULL,
	encryption_header BYTES(1024) NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id),
INTERLEAVE IN PARENT users ON DELETE CASCADE;
`

type Key struct {
	UserId     string
	SessionId  string
	PrivateKey string
	PinToken   string
}
