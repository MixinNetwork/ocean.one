package models

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/dgrijalva/jwt-go"
)

func AuthenticateWithToken(ctx context.Context, jwtToken string) (*User, error) {
	var user *User
	var queryErr error
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
			queryErr = session.TransactionError(ctx, err)
			return nil, queryErr
		} else if u == nil {
			return nil, nil
		}
		user = u

		s, err := readSession(ctx, txn, user.UserId, sid)
		if err != nil {
			queryErr = session.TransactionError(ctx, err)
			return nil, queryErr
		} else if s == nil {
			return nil, nil
		} else if user.MixinId.Valid && s.Code.Valid {
			queryErr = session.TwoFARequiredError(ctx)
			return nil, queryErr
		}
		user.SessionId = s.SessionId

		k, err := readKey(ctx, txn, user.UserId)
		if err != nil {
			queryErr = session.TransactionError(ctx, err)
			return nil, queryErr
		}
		user.Key = k

		pkix, err := hex.DecodeString(s.Secret)
		if err != nil {
			return nil, err
		}
		return x509.ParsePKIXPublicKey(pkix)
	})

	if queryErr != nil {
		return nil, queryErr
	}
	if err == nil && token.Valid {
		return user, nil
	}
	return nil, nil
}
