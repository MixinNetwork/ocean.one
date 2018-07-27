package models

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/uuid"
	"google.golang.org/api/iterator"
)

var sessionColumnsFull = []string{"user_id", "session_id", "secret", "remote_address", "active_at", "created_at"}

func (s *Session) valuesFull() []interface{} {
	return []interface{}{s.UserId, s.SessionId, s.Secret, s.RemoteAddress, s.ActiveAt, s.CreatedAt}
}

type Session struct {
	UserId        string
	SessionId     string
	Secret        string
	RemoteAddress string
	ActiveAt      time.Time
	CreatedAt     time.Time
}

func addSession(ctx context.Context, txn *spanner.ReadWriteTransaction, userId string, secret string) (*Session, error) {
	createdAt := time.Now()
	s := &Session{
		UserId:        userId,
		SessionId:     uuid.NewV4().String(),
		Secret:        secret,
		RemoteAddress: session.RemoteAddress(ctx),
		ActiveAt:      createdAt,
		CreatedAt:     createdAt,
	}
	return s, txn.BufferWrite([]*spanner.Mutation{spanner.Insert("sessions", sessionColumnsFull, s.valuesFull())})
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
