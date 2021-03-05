package models

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/session"
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
	user, err := CreateOrResetUser(ctx, pv.VerificationId, "http://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)
	user, err = CreateSession(ctx, phone, "http://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)
	assert.False(user.MixinId.Valid)
	sess, err := readTestSession(ctx, user)
	assert.Nil(err)
	assert.NotNil(sess)
	assert.False(sess.Code.Valid)
	err = updateUserMixinId(ctx, user)
	assert.Nil(err)
	user, err = CreateSession(ctx, phone, "http://localhost", sessionSecret)
	assert.Nil(err)
	assert.NotNil(user)
	assert.True(user.MixinId.Valid)
	sess, err = readTestSession(ctx, user)
	assert.Nil(err)
	assert.NotNil(sess)
	assert.True(sess.Code.Valid)
	err = VerifySession(ctx, user.UserId, user.SessionId, sess.Code.StringVal)
	assert.Nil(err)
	sess, err = readTestSession(ctx, user)
	assert.Nil(err)
	assert.NotNil(sess)
	assert.False(sess.Code.Valid)
}

func updateUserMixinId(ctx context.Context, user *User) error {
	user.MixinId = spanner.NullString{uuid.NewV4().String() + ":" + "10000", true}
	return session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Update("users", []string{"user_id", "mixin_id"}, []interface{}{user.UserId, user.MixinId}),
	}, "users", "UPDATE", "ConnectMixin")
}

func readTestSession(ctx context.Context, user *User) (*Session, error) {
	txn := session.Database(ctx).ReadOnlyTransaction()
	defer txn.Close()

	return readSession(ctx, txn, user.UserId, user.SessionId)
}
