package models

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/dgrijalva/jwt-go"
)

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

		uid, sid := fmt.Sprint(claims["uid"]), fmt.Sprint(claims["sid"])
		txn := session.Database(ctx).ReadOnlyTransaction()
		defer txn.Close()

		u, err := readUser(ctx, txn, uid)
		if err != nil {
			return nil, session.TransactionError(ctx, err)
		} else if u == nil {
			return nil, nil
		}
		user = u

		s, err := readSession(ctx, txn, sid)
		if err != nil {
			return nil, session.TransactionError(ctx, err)
		} else if s == nil {
			return nil, nil
		}

		pkix, err := hex.DecodeString(s.Secret)
		if err != nil {
			return nil, err
		}
		return x509.ParsePKIXPublicKey(pkix)
	})

	if err != nil && strings.Contains(err.Error(), "spanner") {
		return nil, err
	}
	if err == nil && token.Valid {
		return user, nil
	}
	return nil, nil
}
