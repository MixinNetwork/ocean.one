package durable

import (
	"time"

	"cloud.google.com/go/spanner"
	"github.com/newrelic/go-agent"
	"golang.org/x/net/context"
)

type Database struct {
	spanner     *spanner.Client
	transaction newrelic.Transaction
}

type Transaction interface {
	ReadUsingIndex(context.Context, string, string, spanner.KeySet, []string) *spanner.RowIterator
	Read(context.Context, string, spanner.KeySet, []string) *spanner.RowIterator
	ReadRow(context.Context, string, spanner.Key, []string) (*spanner.Row, error)
	Query(context.Context, spanner.Statement) *spanner.RowIterator
}

func WrapDatabase(client *spanner.Client, transaction newrelic.Transaction) *Database {
	return &Database{client, transaction}
}

func (db *Database) Apply(ctx context.Context, ms []*spanner.Mutation, collection, operation, query string) error {
	if db.transaction != nil {
		ds := db.profile(collection, operation, query)
		defer ds.End()
	}
	_, err := db.spanner.Apply(ctx, ms)
	return err
}

func (db *Database) Query(ctx context.Context, statement spanner.Statement, collection, query string) *spanner.RowIterator {
	if db.transaction != nil {
		ds := db.profile(collection, "SELECT", query)
		defer ds.End()
	}
	return db.spanner.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, statement)
}

func (db *Database) ReadOnlyTransaction() *spanner.ReadOnlyTransaction {
	return db.spanner.ReadOnlyTransaction()
}

func (db *Database) ReadRow(ctx context.Context, table string, key spanner.Key, columns []string, query string) (*spanner.Row, error) {
	if db.transaction != nil {
		ds := db.profile(table, "SELECT", query)
		defer ds.End()
	}
	return db.spanner.Single().ReadRow(ctx, table, key, columns)
}

func (db *Database) Read(ctx context.Context, table string, keys spanner.KeySet, columns []string, query string) *spanner.RowIterator {
	if db.transaction != nil {
		ds := db.profile(table, "SELECT", query)
		defer ds.End()
	}
	return db.spanner.Single().Read(ctx, table, keys, columns)
}

func (db *Database) ReadUsingIndex(ctx context.Context, table, index string, keys spanner.KeySet, columns []string, query string) *spanner.RowIterator {
	if db.transaction != nil {
		ds := db.profile(table, "SELECT", query)
		defer ds.End()
	}
	return db.spanner.Single().ReadUsingIndex(ctx, table, index, keys, columns)
}

func (db *Database) ReadWriteTransaction(ctx context.Context, impl func(context.Context, *spanner.ReadWriteTransaction) error, collection, operation, query string) (time.Time, error) {
	if db.transaction != nil {
		ds := db.profile(collection, operation, query)
		defer ds.End()
	}
	return db.spanner.ReadWriteTransaction(ctx, impl)
}

func OpenSpannerClient(ctx context.Context, name string) (*spanner.Client, error) {
	return spanner.NewClientWithConfig(ctx, name, spanner.ClientConfig{NumChannels: 4,
		SessionPoolConfig: spanner.SessionPoolConfig{
			HealthCheckInterval: 5 * time.Second,
		},
	})
}

func (db *Database) Close() {
	db.spanner.Close()
}

func (db *Database) profile(collection, operation, query string) *newrelic.DatastoreSegment {
	return &newrelic.DatastoreSegment{
		StartTime:          newrelic.StartSegmentNow(db.transaction),
		Product:            newrelic.DatastorePostgres,
		Collection:         collection,
		Operation:          operation,
		ParameterizedQuery: query,
		Host:               "cluster",
		PortPathOrID:       "7777",
		DatabaseName:       "mixin",
	}
}
