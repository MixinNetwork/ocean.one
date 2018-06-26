package persistence

import (
	"cloud.google.com/go/spanner"
)

type Spanner struct {
	spanner *spanner.Client
}

func CreateSpanner(client *spanner.Client) Persist {
	return &Spanner{
		spanner: client,
	}
}
