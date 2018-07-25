package models

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"google.golang.org/api/iterator"
)

var sessionColumnsFull = []string{"user_id", "session_id", "secret", "remote_address", "active_at", "created_at"}

const sessions_DDL = `
CREATE TABLE sessions (
	user_id	          STRING(36) NOT NULL,
	session_id        STRING(36) NOT NULL,
	secret            STRING(512) NOT NULL,
	remote_address    STRING(1024) NOT NULL,
	active_at         TIMESTAMP NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id, session_id),
INTERLEAVE IN PARENT users ON DELETE CASCADE;
`

type Session struct {
	UserId        string
	SessionId     string
	Secret        string
	RemoteAddress string
	ActiveAt      time.Time
	CreatedAt     time.Time
}

func readSession(ctx context.Context, txn durable.Transaction, sid string) (*Session, error) {
	it := txn.Read(ctx, "sessions", spanner.Key{sid}, sessionColumnsFull)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return sessionFromRow(row)
}

func sessionFromRow(row *spanner.Row) (*Session, error) {
	var s Session
	err := row.Columns(&s.UserId, &s.SessionId, &s.Secret, &s.RemoteAddress, &s.ActiveAt, &s.CreatedAt)
	return &s, err
}
