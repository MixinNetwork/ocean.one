package models

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"time"

	"cloud.google.com/go/spanner"
	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

var sessionColumnsFull = []string{"user_id", "session_id", "secret", "code", "remote_address", "active_at", "created_at"}

func (s *Session) valuesFull() []interface{} {
	return []interface{}{s.UserId, s.SessionId, s.Secret, s.Code, s.RemoteAddress, s.ActiveAt, s.CreatedAt}
}

type Session struct {
	UserId        string
	SessionId     string
	Secret        string
	RemoteAddress string
	Code          spanner.NullString
	ActiveAt      time.Time
	CreatedAt     time.Time
}

func CreateSession(ctx context.Context, receiver, password string, secret string) (*User, error) {
	pkix, err := hex.DecodeString(secret)
	if err != nil {
		return nil, session.BadDataError(ctx)
	}
	_, err = x509.ParsePKIXPublicKey(pkix)
	if err != nil {
		return nil, session.BadDataError(ctx)
	}

	var user *User
	_, err = session.Database(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		u, err := readUserByPhone(ctx, txn, receiver)
		if err != nil {
			return err
		}
		if u == nil {
			u, err = readUserByEmail(ctx, txn, receiver)
			if err != nil {
				return err
			}
		}
		if u == nil {
			return session.AuthorizationError(ctx)
		}
		err = bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password))
		if err != nil {
			return session.AuthorizationError(ctx)
		}
		s, err := addSession(ctx, txn, u, secret)
		if err != nil {
			return err
		}
		u.SessionId = s.SessionId
		u.ActiveAt = s.ActiveAt
		u.CreatedAt = s.CreatedAt
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("users", []string{"user_id", "active_at", "created_at"}, []interface{}{u.UserId, u.ActiveAt, u.CreatedAt}),
		})
		user = u
		return nil
	}, "sessions", "INSERT", "CreateSession")

	if err != nil {
		if se, ok := session.ParseError(spanner.ErrDesc(err)); ok {
			return nil, se
		}
		return nil, session.TransactionError(ctx, err)
	}
	return user, nil
}

func VerifySession(ctx context.Context, uid, sid, code string) error {
	_, err := session.Database(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		s, err := readSession(ctx, txn, uid, sid)
		if err != nil {
			return err
		}
		if s == nil || s.Code.StringVal != code {
			return session.TwoFAInvalidError(ctx)
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("sessions", []string{"user_id", "session_id", "code"}, []interface{}{uid, sid, spanner.NullString{"", false}}),
		})
	}, "sessions", "UPDATE", "VerifySession")
	if err != nil {
		if se, ok := session.ParseError(spanner.ErrDesc(err)); ok {
			return se
		}
		return session.TransactionError(ctx, err)
	}
	return nil
}

func addSession(ctx context.Context, txn *spanner.ReadWriteTransaction, user *User, secret string) (*Session, error) {
	createdAt := time.Now()
	s := &Session{
		UserId:        user.UserId,
		SessionId:     uuid.NewV4().String(),
		Secret:        secret,
		RemoteAddress: session.RemoteAddress(ctx),
		ActiveAt:      createdAt,
		CreatedAt:     createdAt,
	}
	if user.MixinId.Valid {
		code, err := generateSixDigitCode(ctx)
		if err != nil {
			return nil, err
		}
		s.Code = spanner.NullString{code, true}
	}
	err := txn.BufferWrite([]*spanner.Mutation{spanner.Insert("sessions", sessionColumnsFull, s.valuesFull())})
	if err != nil {
		return nil, err
	}
	go func() {
		id := bot.UniqueConversationId(config.ClientId, user.UserId)
		data := base64.StdEncoding.EncodeToString([]byte(s.Code.StringVal))
		bot.PostMessage(ctx, id, user.MixinUserId(), uuid.NewV4().String(), "PLAIN_TEXT", data, config.ClientId, config.SessionId, config.SessionKey)
	}()
	return s, nil
}

func readSession(ctx context.Context, txn durable.Transaction, uid, sid string) (*Session, error) {
	it := txn.Read(ctx, "sessions", spanner.Key{uid, sid}, sessionColumnsFull)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return sessionFromRow(row)
}

func cleanupSessions(ctx context.Context, txn *spanner.ReadWriteTransaction, uid string) error {
	stmt := spanner.Statement{
		SQL:    "SELECT session_id FROM sessions WHERE user_id=@user_id LIMIT 1000",
		Params: map[string]interface{}{"user_id": uid},
	}
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var keySets []spanner.KeySet
	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil
		}

		var sid string
		if err := row.Columns(&sid); err != nil {
			return err
		}
		keySets = append(keySets, spanner.Key{uid, sid})
	}

	if len(keySets) > 0 {
		return txn.BufferWrite([]*spanner.Mutation{spanner.Delete("sessions", spanner.KeySets(keySets...))})
	}
	return nil
}

func sessionFromRow(row *spanner.Row) (*Session, error) {
	var s Session
	err := row.Columns(&s.UserId, &s.SessionId, &s.Secret, &s.Code, &s.RemoteAddress, &s.ActiveAt, &s.CreatedAt)
	return &s, err
}
