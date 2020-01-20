package models

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/api/iterator"
)

var keysColumnsFull = []string{"user_id", "session_id", "session_key", "pin_token", "encrypted_pin", "encryption_header", "ocean_key", "created_at"}

func (k *Key) valuesFull() []interface{} {
	return []interface{}{k.UserId, k.SessionId, k.SessionKey, k.PinToken, k.EncryptedPIN, k.EncryptionHeader, k.OceanKey, k.CreatedAt}
}

type Key struct {
	UserId           string
	SessionId        string
	SessionKey       string
	PinToken         string
	EncryptedPIN     string
	EncryptionHeader []byte
	OceanKey         string
	CreatedAt        time.Time

	DecryptedPIN string
}

func (k *Key) MixinToken(ctx context.Context, uri string) (string, error) {
	sum := sha256.Sum256([]byte("GET" + uri))
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.MapClaims{
		"uid": k.UserId,
		"sid": k.SessionId,
		"scp": "ASSETS:READ SNAPSHOTS:READ",
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"sig": hex.EncodeToString(sum[:]),
	})

	block, _ := pem.Decode([]byte(k.SessionKey))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", session.ServerError(ctx, err)
	}
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", session.ServerError(ctx, err)
	}
	return tokenString, nil
}

func (k *Key) OceanToken(ctx context.Context) (string, error) {
	oceanKey, err := hex.DecodeString(k.OceanKey)
	if err != nil {
		return "", session.ServerError(ctx, err)
	}
	privateKey, err := x509.ParseECPrivateKey(oceanKey)
	if err != nil {
		return "", session.ServerError(ctx, err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"uid": k.UserId,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", session.ServerError(ctx, err)
	}
	return tokenString, nil
}

func consumePoolKey(ctx context.Context, txn *spanner.ReadWriteTransaction) (*Key, error) {
	it := txn.Query(ctx, spanner.Statement{
		SQL: fmt.Sprintf("SELECT %s FROM pool_keys LIMIT 1", strings.Join(poolKeysColumnsFull, ",")),
	})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	pk, err := poolKeyFromRow(row)
	if err != nil {
		return nil, err
	}

	return &Key{
		UserId:           pk.UserId,
		SessionId:        pk.SessionId,
		SessionKey:       pk.SessionKey,
		PinToken:         pk.PinToken,
		EncryptedPIN:     pk.EncryptedPIN,
		EncryptionHeader: pk.EncryptionHeader,
		OceanKey:         pk.OceanKey,
		CreatedAt:        time.Now(),
	}, txn.BufferWrite([]*spanner.Mutation{spanner.Delete("pool_keys", spanner.Key{pk.UserId})})
}

func readKey(ctx context.Context, txn durable.Transaction, userId string) (*Key, error) {
	it := txn.Read(ctx, "keys", spanner.Key{userId}, keysColumnsFull)
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	key, err := keyFromRow(row)
	if err != nil {
		return nil, err
	}

	privateBlock, _ := pem.Decode([]byte(config.AssetPrivateKey))
	privateKey, err := x509.ParsePKCS1PrivateKey(privateBlock.Bytes)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, key.EncryptionHeader[encryptionHeaderLength:], nil)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(key.EncryptedPIN)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	iv := cipherBytes[:aes.BlockSize]
	source := cipherBytes[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(source, source)

	length := len(source)
	unpadding := int(source[length-1])
	key.DecryptedPIN = string(source[:length-unpadding])
	return key, nil
}

func keyFromRow(row *spanner.Row) (*Key, error) {
	var k Key
	err := row.Columns(&k.UserId, &k.SessionId, &k.SessionKey, &k.PinToken, &k.EncryptedPIN, &k.EncryptionHeader, &k.OceanKey, &k.CreatedAt)
	return &k, err
}

func (k *Key) sendTransfer(ctx context.Context, recipientId, assetId string, amount number.Decimal, traceId, memo string) error {
	return bot.CreateTransfer(ctx, &bot.TransferInput{
		AssetId:     assetId,
		RecipientId: recipientId,
		Amount:      amount,
		TraceId:     traceId,
		Memo:        memo,
	}, k.UserId, k.SessionId, k.SessionKey, k.DecryptedPIN, k.PinToken)
}
