package persistence

import (
	"context"

	"cloud.google.com/go/spanner"
)

type contextValueKey int

const (
	keySpanner contextValueKey = 1
)

func SetupSpanner(ctx context.Context, client *spanner.Client) context.Context {
	return context.WithValue(ctx, keySpanner, client)
}

func Spanner(ctx context.Context) *spanner.Client {
	v, _ := ctx.Value(keySpanner).(*spanner.Client)
	return v
}
