package services

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

func standardServiceHealth(ctx context.Context) error {
	stmt := spanner.Statement{SQL: "SELECT user_id FROM users LIMIT 1"}
	it := session.Database(ctx).Query(ctx, stmt, "users", "standardServiceHealth")
	defer it.Stop()
	if _, err := it.Next(); err != nil && err != iterator.Done {
		return session.TransactionError(ctx, err)
	}
	return nil
}
