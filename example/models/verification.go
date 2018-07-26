package models

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/MixinNetwork/ocean.one/example/uuid"
	"github.com/MixinNetwork/ocean.one/example/wire"
	"google.golang.org/api/iterator"
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
	if category != VerificationCategoryPhone {
		return nil, session.BadDataError(ctx)
	}
	if phone, err := ValidatePhoneNumberFormat(ctx, receiver); err != nil {
		return nil, err
	} else {
		receiver = phone
	}

	code, err := generateSixDigitCode(ctx)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}

	if config.RecaptchaEnable {
		if recaptcha == "" {
			return nil, session.RecaptchaRequiredError(ctx)
		}
		if success, err := verifyRecaptcha(ctx, recaptcha); err != nil {
			return nil, session.BadDataError(ctx)
		} else if !success {
			return nil, session.RecaptchaVerifyError(ctx)
		}
	}

	vf := &Verification{
		VerificationId: uuid.NewV4().String(),
		Category:       category,
		Receiver:       receiver,
		Code:           code,
		Provider:       wire.SMSProviderTelesign,
		CreatedAt:      time.Now(),
	}

	shouldDeliver := true
	_, err = session.Database(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		last, err := checkVerificationFrequency(ctx, txn, vf)
		if err != nil {
			return err
		}
		if last != nil && last.CreatedAt.After(time.Now().UTC().Add(-1*config.SMSDeliveryInterval)) {
			vf, shouldDeliver = last, false
			return nil
		}
		if last != nil && last.Provider == wire.SMSProviderTelesign {
			vf.Provider = wire.SMSProviderTwilio
		}
		txn.BufferWrite([]*spanner.Mutation{spanner.Insert("verifications", verificationsColumnsFull, vf.valuesFull())})
		return nil
	}, "verifications", "INSERT", "CreateVerification")

	if err != nil {
		if sessionErr, ok := session.ParseError(spanner.ErrDesc(err)); ok {
			return nil, sessionErr
		}
		return nil, session.TransactionError(ctx, err)
	}

	if shouldDeliver {
		limiter := session.Limiter(ctx)
		limiterKey := "limiter:code:receiver:" + vf.Receiver
		limiterRetry := config.PhoneSMSSenderLimit
		limiterDuration := time.Hour * 24
		limiterCount, err := limiter.Available(limiterKey, limiterDuration, limiterRetry, true)
		if err != nil {
			return nil, session.ServerError(ctx, err)
		} else if limiterCount < 1 {
			return vf, nil
		}
		if err := wire.SendVerificationCode(vf.Provider, vf.Receiver, vf.Code); err != nil {
			session.PhoneSMSDeliveryError(ctx, vf.Receiver, err)
		}
	}

	return vf, nil
}

func DoVerification(ctx context.Context, id, code string) (*Verification, error) {
	return nil, nil
}

func checkVerificationFrequency(ctx context.Context, txn durable.Transaction, vf *Verification) (*Verification, error) {
	query := fmt.Sprintf("SELECT %s FROM verifications@{FORCE_INDEX=verifications_by_receiver_created_at_desc} WHERE receiver=@receiver ORDER BY receiver, created_at DESC LIMIT 1", strings.Join(verificationsColumnsFull, ","))
	statement := spanner.Statement{SQL: query, Params: map[string]interface{}{"receiver": vf.Receiver}}
	it := txn.Query(ctx, statement)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	verification, err := verificationFromRow(row)
	if err != nil {
		return nil, err
	}
	return verification, nil
}

func verificationFromRow(row *spanner.Row) (*Verification, error) {
	var v Verification
	err := row.Columns(&v.VerificationId, &v.Category, &v.Receiver, &v.Code, &v.Provider, &v.CreatedAt, &v.VerifiedAt)
	return &v, err
}

func generateSixDigitCode(ctx context.Context) (string, error) {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", session.ServerError(ctx, err)
	}
	c := binary.LittleEndian.Uint64(b[:]) % 1000000
	if c < 100000 {
		c = 100000 + c
	}
	return fmt.Sprint(c), nil
}
