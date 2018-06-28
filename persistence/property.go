package persistence

import (
	"time"
)

type Property struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}
