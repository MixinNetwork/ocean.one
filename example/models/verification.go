package models

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

const (
	VerificationCategoryPhone = "PHONE"
	VerificationCategoryEmail = "EMAIL"
)

var verificationsColumnsFull = []string{"verification_id", "category", "receiver", "code", "provider", "created_at", "verified_at"}

func (p *Verification) valuesFull() []interface{} {
	return []interface{}{p.VerificationId, p.Category, p.Receiver, p.Code, p.Provider, p.CreatedAt, p.VerifiedAt}
}

type Verification struct {
	VerificationId string
	Category       string
	Receiver       string
	Code           string
	Provider       string
	CreatedAt      time.Time
	VerifiedAt     spanner.NullTime
}

func CreateVerification(ctx context.Context, category, receiver string, recaptcha string) (*Verification, error) {
	return nil, nil
}

func DoVerification(ctx context.Context, id, code string) (*Verification, error) {
	return nil, nil
}
