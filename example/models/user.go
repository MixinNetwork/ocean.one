package models

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/api/iterator"
)

const users_DDL = `
CREATE TABLE users (
	user_id	           STRING(36) NOT NULL,
	email              STRING(512),
	phone              STRING(512),
	mixin_id           STRING(36),
	identity_id        STRING(36),
	full_name          STRING(512) NOT NULL,
	encrypted_password STRING(1024) NOT NULL,
	session_secret     STRING(512) NOT NULL,
	active_at          TIMESTAMP NOT NULL,
	created_at         TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id);
`

var usersColumnsFull = []string{"user_id", "email", "phone", "mixin_id", "identity_id", "full_name", "encrypted_password", "session_secret", "active_at", "created_at"}

func (u *User) valuesFull() []interface{} {
	return []interface{}{u.UserId, u.Email, u.Phone, u.MixinId, u.FullName, u.EncryptedPassword, u.SessionSecret, u.ActiveAt, u.CreatedAt}
}

type User struct {
	UserId            string
	Email             spanner.NullString
	Phone             spanner.NullString
	MixinId           spanner.NullString
	IdentityId        spanner.NullString
	FullName          string
	EncryptedPassword string
	SessionSecret     string
	ActiveAt          time.Time
	CreatedAt         time.Time
}

func AuthenticateWithToken(ctx context.Context, jwtToken string) (*User, error) {
	var user *User
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, nil
		}
		_, ok = token.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, nil
		}

		userId := fmt.Sprint(claims["uid"])
		it := session.Database(ctx).Read(ctx, "users", spanner.Key{userId}, usersColumnsFull, "Authenticate")
		defer it.Stop()

		row, err := it.Next()
		if err == iterator.Done {
			return nil, nil
		} else if err != nil {
			return nil, session.TransactionError(ctx, err)
		}

		user, err = userFromRow(row)
		if err != nil {
			return nil, session.TransactionError(ctx, err)
		}
		return hex.DecodeString(user.SessionSecret)
	})

	if err != nil && strings.Contains(err.Error(), "spanner") {
		return nil, err
	}
	if err == nil && token.Valid {
		return user, nil
	}
	return nil, nil
}

func userFromRow(row *spanner.Row) (*User, error) {
	var u User
	err := row.Columns(&u.UserId, &u.Email, &u.Phone, &u.MixinId, &u.IdentityId, &u.FullName, &u.EncryptedPassword, &u.SessionSecret, &u.ActiveAt, &u.CreatedAt)
	return &u, err
}
