package models

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

type Property struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

func readProperty(ctx context.Context, txn durable.Transaction, key string) (string, error) {
	it := txn.Read(ctx, "properties", spanner.Key{key}, []string{"value"})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return "", nil
	} else if err != nil {
		return "", err
	}

	var value string
	err = row.Column(0, &value)
	return value, err
}

func ReadProperty(ctx context.Context, key string) (string, error) {
	txn := session.Database(ctx).ReadOnlyTransaction()
	defer txn.Close()
	value, err := readProperty(ctx, txn, key)
	if err != nil {
		return "", session.TransactionError(ctx, err)
	}
	return value, nil
}

func writeProperty(ctx context.Context, txn *spanner.ReadWriteTransaction, key, value string) error {
	return txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertOrUpdate("properties", []string{"key", "value", "updated_at"}, []interface{}{key, value, time.Now()}),
	})
}

func WriteProperty(ctx context.Context, key, value string) error {
	_, err := session.Database(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return writeProperty(ctx, txn, key, value)
	}, "properties", "UPDATE", "WriteProperty")

	if err != nil {
		return session.TransactionError(ctx, err)
	}
	return nil
}
