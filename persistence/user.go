package persistence

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

type User struct {
	UserId    string
	PublicKey string
}

func UpdateUserPublicKey(ctx context.Context, userId, publicKey string) error {
	if _, err := hex.DecodeString(publicKey); err != nil {
		return nil
	}

	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		it := txn.ReadUsingIndex(ctx, "users", "users_by_public_key", spanner.Key{publicKey}, []string{"user_id"})
		defer it.Stop()

		_, err := it.Next()
		if err == iterator.Done {
		} else if err != nil {
			return err
		} else {
			return nil
		}

		txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("users", map[string]interface{}{
			"user_id":    userId,
			"public_key": publicKey,
		})})
		return nil
	})

	return err
}

func Authenticate(ctx context.Context, jwtToken string) (string, error) {
	var userId string
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, nil
		}
		_, ok = token.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, nil
		}
		id, _ := uuid.FromString(fmt.Sprint(claims["uid"]))
		if id.String() == uuid.Nil.String() {
			return nil, nil
		} else {
			userId = id.String()
		}

		it := Spanner(ctx).Single().Read(ctx, "users", spanner.Key{userId}, []string{"public_key"})
		defer it.Stop()
		row, err := it.Next()
		if err == iterator.Done {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		var publicKey string
		err = row.Columns(&publicKey)
		if err != nil {
			return nil, err
		}

		return hex.DecodeString(publicKey)
	})

	if err != nil && strings.Contains(err.Error(), "spanner") {
		return "", err
	}
	if err == nil && token.Valid {
		return userId, nil
	}
	return "", nil
}
