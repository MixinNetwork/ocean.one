package persistence

import (
	"context"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type User struct {
	UserId    string
	PublicKey string
}

func UpdateUserPublicKey(ctx context.Context, userId, publicKey string) error {
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
