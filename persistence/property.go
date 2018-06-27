package persistence

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type Property struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

func ReadProperty(ctx context.Context, key string) (string, error) {
	it := Spanner(ctx).Single().Read(ctx, "properties", spanner.Key{key}, []string{"value"})
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

func WriteProperty(ctx context.Context, key, value string) error {
	_, err := Spanner(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdate("properties", []string{"key", "value", "updated_at"}, []interface{}{key, value, time.Now()}),
	})
	return err
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
