package models

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

type Property struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

func ReadProperty(ctx context.Context, key string) (string, error) {
	it := session.Database(ctx).Read(ctx, "properties", spanner.Key{key}, []string{"value"}, "ReadProperty")
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return "", nil
	} else if err != nil {
		return "", session.TransactionError(ctx, err)
	}

	var value string
	err = row.Column(0, &value)
	if err != nil {
		return "", session.TransactionError(ctx, err)
	}
	return value, nil
}

func WriteProperty(ctx context.Context, key, value string) error {
	err := session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdate("properties", []string{"key", "value", "updated_at"}, []interface{}{key, value, time.Now()}),
	}, "properties", "INSERT", "WriteProperty")
	if err != nil {
		return session.TransactionError(ctx, err)
	}
	return nil
}

func ReadPropertyAsTime(ctx context.Context, key string) (time.Time, error) {
	var offset time.Time
	timestamp, err := ReadProperty(ctx, key)
	if err != nil {
		return offset, err
	}
	if timestamp != "" {
		return time.Parse(time.RFC3339Nano, timestamp)
	}
	return offset, nil
}

func WriteTimeProperty(ctx context.Context, key string, value time.Time) error {
	return WriteProperty(ctx, key, value.UTC().Format(time.RFC3339Nano))
}
