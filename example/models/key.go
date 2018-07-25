package models

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	mathRand "math/rand"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

const encryptionHeaderLength = 16

const keys_DDL = `
CREATE TABLE keys (
	user_id	          STRING(36) NOT NULL,
	session_id        STRING(36) NOT NULL,
	session_key       STRING(1024) NOT NULL,
	pin_token         STRING(512) NOT NULL,
	encrypted_pin     STRING(512) NOT NULL,
	encryption_header BYTES(1024) NOT NULL,
	created_at        TIMESTAMP NOT NULL,
) PRIMARY KEY(user_id),
INTERLEAVE IN PARENT users ON DELETE CASCADE;
`

var keysColumnsFull = []string{"user_id", "session_id", "session_key", "pin_token", "encrypted_pin", "encryption_header", "created_at"}

func (k *Key) valuesFull() []interface{} {
	return []interface{}{k.UserId, k.SessionId, k.SessionKey, k.PinToken, k.EncryptedPIN, k.EncryptionHeader, k.CreatedAt}
}

type Key struct {
	UserId           string
	SessionId        string
	SessionKey       string
	PinToken         string
	EncryptedPIN     string
	EncryptionHeader []byte
	CreatedAt        time.Time

	DecryptedPIN string
}

func GenerateKey(ctx context.Context) (*Key, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	sessionSecret := base64.StdEncoding.EncodeToString(publicKeyBytes)
	data, err := json.Marshal(map[string]string{
		"session_secret": sessionSecret,
	})
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	token, err := bot.SignAuthenticationToken(config.ClientId, config.SessionId, config.SessionKey, "POST", "/users", string(data))
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	body, err := bot.Request(ctx, "POST", "/pin/update", data, token)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	var resp struct {
		Data struct {
			UserId    string `json:"user_id"`
			SessionId string `json:"session_id"`
			PinToken  string `json:"pin_token"`
		} `json:"data"`
		Error bot.Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	if resp.Error.Code > 0 {
		return nil, session.ServerError(ctx, errors.New(resp.Error.Description))
	}
	key := &Key{
		UserId:    resp.Data.UserId,
		SessionId: resp.Data.SessionId,
		SessionKey: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})),
		PinToken:  resp.Data.PinToken,
		CreatedAt: time.Now(),
	}
	err = key.setupPIN(ctx)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	err = session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Insert("keys", keysColumnsFull, key.valuesFull()),
	}, "keys", "INSERT", "GenerateKey")
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return key, nil
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

	privateBytes, err := base64.StdEncoding.DecodeString(config.AssetPrivateKey)
	if err != nil {
		return nil, session.ServerError(ctx, err)
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateBytes)
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
	err := row.Columns(&k.UserId, &k.SessionId, &k.SessionKey, &k.PinToken, &k.EncryptedPIN, &k.EncryptionHeader, &k.CreatedAt)
	return &k, err
}

func (k *Key) setupPIN(ctx context.Context) error {
	mathRand.Seed(time.Now().UnixNano())
	pin := fmt.Sprintf("%d%d%d%d%d%d", mathRand.Intn(10), mathRand.Intn(10), mathRand.Intn(10), mathRand.Intn(10), mathRand.Intn(10), mathRand.Intn(10))
	encryptedPIN := mixinEncryptPIN(ctx, pin, k.PinToken, k.SessionId, k.SessionKey, uint64(time.Now().UnixNano()))
	data, _ := json.Marshal(map[string]string{"pin": encryptedPIN})

	token, err := bot.SignAuthenticationToken(k.UserId, k.SessionId, k.SessionKey, "POST", "/pin/update", string(data))
	if err != nil {
		return session.ServerError(ctx, err)
	}
	body, err := bot.Request(ctx, "POST", "/pin/update", data, token)
	if err != nil {
		return session.ServerError(ctx, err)
	}
	var resp struct {
		Error bot.Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return session.ServerError(ctx, err)
	}
	if resp.Error.Code > 0 {
		return session.ServerError(ctx, errors.New(resp.Error.Description))
	}
	encryptedPIN, encryptionHeader, err := encryptPIN(ctx, pin)
	if err != nil {
		return session.ServerError(ctx, err)
	}
	k.EncryptedPIN = encryptedPIN
	k.EncryptionHeader = encryptionHeader
	return nil
}

func (k *Key) sendTransfer(ctx context.Context, recipientId, assetId string, amount number.Decimal, traceId, memo string) error {
	if amount.Exhausted() {
		return nil
	}

	pin := mixinEncryptPIN(ctx, k.DecryptedPIN, k.PinToken, k.SessionId, k.SessionKey, uint64(time.Now().UnixNano()))
	data, err := json.Marshal(map[string]interface{}{
		"asset_id":    assetId,
		"opponent_id": recipientId,
		"amount":      amount.Persist(),
		"pin":         pin,
		"trace_id":    traceId,
		"memo":        memo,
	})
	if err != nil {
		return err
	}

	token, err := bot.SignAuthenticationToken(k.UserId, k.SessionId, k.SessionKey, "POST", "/transfers", string(data))
	if err != nil {
		return err
	}
	body, err := bot.Request(ctx, "POST", "/transfers", data, token)
	if err != nil {
		return err
	}

	var resp struct {
		Error bot.Error `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return err
	}
	if resp.Error.Code > 0 {
		return errors.New(resp.Error.Description)
	}
	return nil
}

func encryptPIN(ctx context.Context, pin string) (string, []byte, error) {
	aesKey := make([]byte, 32)
	_, err := rand.Read(aesKey)
	if err != nil {
		return "", nil, session.ServerError(ctx, err)
	}
	publicBytes, err := base64.StdEncoding.DecodeString(config.AssetPublicKey)
	if err != nil {
		return "", nil, session.ServerError(ctx, err)
	}
	assetPublicKey, err := x509.ParsePKCS1PublicKey(publicBytes)
	if err != nil {
		return "", nil, session.ServerError(ctx, err)
	}
	aesKeyEncrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, assetPublicKey, aesKey, nil)
	if err != nil {
		return "", nil, session.ServerError(ctx, err)
	}
	encryptionHeader := make([]byte, encryptionHeaderLength)
	encryptionHeader = append(encryptionHeader, aesKeyEncrypted...)

	paddingSize := aes.BlockSize - len(pin)%aes.BlockSize
	paddingBytes := bytes.Repeat([]byte{byte(paddingSize)}, paddingSize)
	plainBytes := append([]byte(pin), paddingBytes...)
	cipherBytes := make([]byte, aes.BlockSize+len(plainBytes))
	iv := cipherBytes[:aes.BlockSize]
	_, err = rand.Read(iv)
	if err != nil {
		return "", nil, session.ServerError(ctx, err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", nil, session.ServerError(ctx, err)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherBytes[aes.BlockSize:], plainBytes)
	return base64.StdEncoding.EncodeToString(cipherBytes), encryptionHeader, nil
}

func mixinEncryptPIN(ctx context.Context, pin, pinToken, sessionId, privateKey string, iterator uint64) string {
	token, _ := base64.StdEncoding.DecodeString(pinToken)
	keyBytes := token[:32]
	pinByte := []byte(pin)
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().Unix()))
	pinByte = append(pinByte, timeBytes...)
	iteratorBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(iteratorBytes, iterator)
	pinByte = append(pinByte, iteratorBytes...)
	padding := aes.BlockSize - len(pinByte)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	pinByte = append(pinByte, padtext...)
	block, _ := aes.NewCipher(keyBytes)
	ciphertext := make([]byte, aes.BlockSize+len(pinByte))
	iv := ciphertext[:aes.BlockSize]
	io.ReadFull(rand.Reader, iv)
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], pinByte)
	return base64.StdEncoding.EncodeToString(ciphertext)
}
