package persistence

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (persist *Spanner) ReadProperty(ctx context.Context, key string) (string, error) {
	it := persist.spanner.Single().Read(ctx, "properties", spanner.Key{key}, []string{"value"})
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

func (persist *Spanner) WriteProperty(ctx context.Context, key, value string) error {
	_, err := persist.spanner.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdate("properties", []string{"key", "value", "updated_at"}, []interface{}{key, value, time.Now()}),
	})
	return err
}

func (persist *Spanner) ReadPropertyAsTime(ctx context.Context, key string) (time.Time, error) {
	var offset time.Time
	timestamp, err := persist.ReadProperty(ctx, key)
	if err != nil {
		return offset, err
	}
	if timestamp != "" {
		return time.Parse(time.RFC3339Nano, timestamp)
	}
	return offset, nil
}

func (persist *Spanner) WriteTimeProperty(ctx context.Context, key string, value time.Time) error {
	return persist.WriteProperty(ctx, key, value.UTC().Format(time.RFC3339Nano))
}
