package models

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"testing"
	"time"

	"github.com/MixinNetwork/ocean.one/example/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserCRUD(t *testing.T) {
	ctx := setupTestContext()
	defer teardownTestContext(ctx)
	assert := assert.New(t)

	for i := 0; i < 2; i++ {
		key := &PoolKey{
			UserId:           uuid.NewV4().String(),
			SessionId:        uuid.NewV4().String(),
			SessionKey:       uuid.NewV4().String(),
			PinToken:         uuid.NewV4().String(),
			EncryptedPIN:     uuid.NewV4().String(),
			EncryptionHeader: []byte{},
			OceanKey:         uuid.NewV4().String(),
			CreatedAt:        time.Now(),
		}
		key.persist(ctx)
	}

	// Reset Password failure
	phone := "+8618678006575"
	pv, err := CreateVerification(ctx, VerificationCategoryPhone, phone, "")
	assert.Nil(err)
	assert.NotNil(pv)
	pv, err = DoVerification(ctx, pv.VerificationId, pv.Code)
	assert.Nil(err)
	assert.NotNil(pv)
	privateKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(privateKey.Public())
	sessionSecret := hex.EncodeToString(publicKeyBytes)
	user, err := ResetPassword(ctx, pv.VerificationId, "http://localhost", sessionSecret)
	assert.NotNil(err)
	assert.Nil(user)
	user, err = CreateUser(ctx, pv.VerificationId, "http://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)
	user, err = CreateSession(ctx, phone, "http://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)

	// Reset Password successfully
	pv, err = CreateVerification(ctx, VerificationCategoryPhone, phone, "")
	assert.Nil(err)
	assert.NotNil(pv)
	pv, err = DoVerification(ctx, pv.VerificationId, pv.Code)
	assert.Nil(err)
	assert.NotNil(pv)
	user, err = ResetPassword(ctx, pv.VerificationId, "https://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)
	user, err = CreateSession(ctx, phone, "http://localhost", sessionSecret)
	assert.NotNil(err)
	assert.Nil(user)
	user, err = CreateSession(ctx, phone, "https://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)
}
