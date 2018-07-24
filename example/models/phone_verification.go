package models

import (
	"time"
)

const phoneVerifications_DDL = `
CREATE TABLE phone_verifications (
	verification_id   STRING(36) NOT NULL,
	phone             STRING(512) NOT NULL,
	code              STRING(128) NOT NULL,
	provider          STRING(128) NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(verification_id);
`

var phoneVerificationsColumnsFull = []string{"verification_id", "phone", "code", "provider", "created_at"}

func (p *PhoneVerification) valuesFull() []interface{} {
	return []interface{}{p.VerificationId, p.Phone, p.Code, p.Provider, p.CreatedAt}
}

type PhoneVerification struct {
	VerificationId string
	Phone          string
	Code           string
	Provider       string
	CreatedAt      time.Time
}
