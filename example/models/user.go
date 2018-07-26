package models

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"google.golang.org/api/iterator"
)

var usersColumnsFull = []string{"user_id", "email", "phone", "mixin_id", "identity_id", "full_name", "encrypted_password", "active_at", "created_at"}

func (u *User) valuesFull() []interface{} {
	return []interface{}{u.UserId, u.Email, u.Phone, u.MixinId, u.FullName, u.EncryptedPassword, u.ActiveAt, u.CreatedAt}
}

type User struct {
	UserId            string
	Email             spanner.NullString
	Phone             spanner.NullString
	MixinId           spanner.NullString
	IdentityId        spanner.NullString
	FullName          string
	EncryptedPassword string
	ActiveAt          time.Time
	CreatedAt         time.Time
}

func CreateUser(ctx context.Context, verficationId, password, sessionSecret string) (*User, error) {
	return nil, nil
}

func readUser(ctx context.Context, txn durable.Transaction, userId string) (*User, error) {
	it := txn.Read(ctx, "users", spanner.Key{userId}, usersColumnsFull)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return userFromRow(row)
}

func userFromRow(row *spanner.Row) (*User, error) {
	var u User
	err := row.Columns(&u.UserId, &u.Email, &u.Phone, &u.MixinId, &u.IdentityId, &u.FullName, &u.EncryptedPassword, &u.ActiveAt, &u.CreatedAt)
	return &u, err
}
