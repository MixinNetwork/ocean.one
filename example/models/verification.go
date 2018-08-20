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
	if category != VerificationCategoryPhone && category != VerificationCategoryEmail {
		return nil, session.BadDataError(ctx)
	}
	provider := wire.SESProviderAWS
	if category == VerificationCategoryPhone {
		provider = wire.SMSProviderTelesign
		if phone, err := ValidatePhoneNumberFormat(ctx, receiver); err != nil {
			return nil, err
		} else {
			receiver = phone
		}
	}
	if category == VerificationCategoryEmail {
		if email, err := ValidateEmailFormat(ctx, receiver); err != nil {
			return nil, err
		} else {
			receiver = email
		}
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
		Provider:       provider,
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
		if category == VerificationCategoryPhone && last != nil && last.Provider == wire.SMSProviderTelesign {
			vf.Provider = wire.SMSProviderTwilio
		}
		txn.BufferWrite([]*spanner.Mutation{spanner.Insert("verifications", verificationsColumnsFull, vf.valuesFull())})
		return nil
	}, "verifications", "INSERT", "CreateVerification")

	if err != nil {
		if se, ok := session.ParseError(spanner.ErrDesc(err)); ok {
			return nil, se
		}
		return nil, session.TransactionError(ctx, err)
	}

	if shouldDeliver {
		limiter := session.Limiter(ctx)
		limiterKey := "limiter:code:receiver:" + vf.Receiver
		limiterRetry := config.VerificationSendLimit
		limiterDuration := time.Hour * 24
		limiterCount, err := limiter.Available(limiterKey, limiterDuration, limiterRetry, true)
		if err != nil {
			return nil, session.ServerError(ctx, err)
		} else if limiterCount < 1 {
			return vf, nil
		}
		wire.SendVerificationCode(ctx, vf.Category, vf.Provider, vf.Receiver, vf.Code)
	}

	return vf, nil
}

func DoVerification(ctx context.Context, id, code string) (*Verification, error) {
	limiter := session.Limiter(ctx)
	limiterRetry := config.VerificationValidateLimit
	limiterDuration := time.Minute * 60

	vf, err := findVerificationById(ctx, id)
	if err != nil {
		return nil, err
	}

	limiterKey := "limiter:verification:receiver:" + vf.Receiver
	limiterCount, err := limiter.Available(limiterKey, limiterDuration, limiterRetry, false)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	} else if limiterCount < 1 {
		return nil, session.TooManyRequestsError(ctx)
	}

	if vf.Code != code {
		vf, err = findVerificationByReceiverAndCode(ctx, vf.Receiver, code)
		if err != nil {
			if se, ok := err.(session.Error); ok && se.Code == 20113 {
				limiterCount, err := limiter.Available(limiterKey, limiterDuration, limiterRetry, true)
				if err != nil {
					return nil, session.ServerError(ctx, err)
				} else if limiterCount < 1 {
					return nil, session.TooManyRequestsError(ctx)
				}
			}
			return nil, err
		}
	}

	if time.Now().After(vf.CreatedAt.Add(time.Minute * 30)) {
		limiterCount, err := limiter.Available(limiterKey, limiterDuration, limiterRetry, true)
		if err != nil {
			return nil, session.ServerError(ctx, err)
		} else if limiterCount < 1 {
			return nil, session.TooManyRequestsError(ctx)
		}
		return nil, session.VerificationCodeExpiredError(ctx)
	}

	err = limiter.Clear(limiterKey, limiterDuration)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}

	vf.VerifiedAt = spanner.NullTime{time.Now(), true}
	err = session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Update("verifications", []string{"verification_id", "verified_at"}, []interface{}{vf.VerificationId, vf.VerifiedAt}),
	}, "verifications", "INSERT", "DoVerification")
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}

	return vf, nil
}

func findVerificationById(ctx context.Context, id string) (*Verification, error) {
	txn := session.Database(ctx).ReadOnlyTransaction()
	defer txn.Close()

	vf, err := readVerification(ctx, txn, id)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	} else if vf == nil {
		return nil, session.VerificationCodeInvalidError(ctx)
	}

	return vf, nil
}

func findVerificationByReceiverAndCode(ctx context.Context, receiver, code string) (*Verification, error) {
	query := fmt.Sprintf("SELECT %s FROM verifications@{FORCE_INDEX=verifications_by_receiver_created_desc} WHERE receiver=@receiver AND code=@code ORDER BY receiver, created_at DESC LIMIT 1", strings.Join(verificationsColumnsFull, ","))
	statement := spanner.Statement{SQL: query, Params: map[string]interface{}{"receiver": receiver, "code": code}}
	it := session.Database(ctx).Query(ctx, statement, "verifications", "findVerificationByReceiverAndCode")
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, session.VerificationCodeInvalidError(ctx)
	} else if err != nil {
		return nil, session.TransactionError(ctx, err)
	}

	vf, err := verificationFromRow(row)
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return vf, nil
}

func checkVerificationFrequency(ctx context.Context, txn durable.Transaction, vf *Verification) (*Verification, error) {
	query := fmt.Sprintf("SELECT %s FROM verifications@{FORCE_INDEX=verifications_by_receiver_created_desc} WHERE receiver=@receiver ORDER BY receiver, created_at DESC LIMIT 1", strings.Join(verificationsColumnsFull, ","))
	statement := spanner.Statement{SQL: query, Params: map[string]interface{}{"receiver": vf.Receiver}}
	it := txn.Query(ctx, statement)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	last, err := verificationFromRow(row)
	if err != nil {
		return nil, err
	}
	return last, nil
}

func readVerification(ctx context.Context, txn durable.Transaction, id string) (*Verification, error) {
	it := txn.Read(ctx, "verifications", spanner.Key{id}, verificationsColumnsFull)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return verificationFromRow(row)
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
