package models

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/session"
	"github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
)

const encryptionHeaderLength = 16

var poolKeysColumnsFull = []string{"user_id", "session_id", "session_key", "pin_token", "encrypted_pin", "encryption_header", "ocean_key", "created_at"}

func (k *PoolKey) valuesFull() []interface{} {
	return []interface{}{k.UserId, k.SessionId, k.SessionKey, k.PinToken, k.EncryptedPIN, k.EncryptionHeader, k.OceanKey, k.CreatedAt}
}

type PoolKey struct {
	UserId           string
	SessionId        string
	SessionKey       string
	PinToken         string
	EncryptedPIN     string
	EncryptionHeader []byte
	OceanKey         string
	CreatedAt        time.Time

	PlainPIN string
}

func GeneratePoolKey(ctx context.Context) (*PoolKey, error) {
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

	body, err := bot.Request(ctx, "POST", "/users", data, token)
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

	key := &PoolKey{
		UserId:    resp.Data.UserId,
		SessionId: resp.Data.SessionId,
		SessionKey: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})),
		PinToken:  resp.Data.PinToken,
		CreatedAt: time.Now(),
	}

	for {
		err := key.setupPIN(ctx)
		if err == nil {
			break
		}
		log.Println(session.ServerError(ctx, err))
		time.Sleep(1 * time.Second)
	}

	for {
		err := key.setupOceanKey(ctx)
		if err == nil {
			break
		}
		log.Println(session.ServerError(ctx, err))
		time.Sleep(1 * time.Second)
	}

	for {
		err := key.persist(ctx)
		if err == nil {
			break
		}
		log.Println(session.TransactionError(ctx, err))
		time.Sleep(500 * time.Millisecond)
	}
	return key, nil
}

func (k *PoolKey) persist(ctx context.Context) error {
	return session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Insert("pool_keys", poolKeysColumnsFull, k.valuesFull()),
	}, "pool_keys", "INSERT", "GeneratePoolKey")
}

func (k *PoolKey) setupOceanKey(ctx context.Context) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return session.ServerError(ctx, err)
	}
	oceanKey, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return session.ServerError(ctx, err)
	}

	pub, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		return session.ServerError(ctx, err)
	}
	sig := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&sig, handle)
	action := map[string][]byte{"U": pub}
	err = encoder.Encode(action)
	if err != nil {
		return session.ServerError(ctx, err)
	}

	err = bot.CreateTransfer(ctx, &bot.TransferInput{
		AssetId:     config.OOOAssetId,
		RecipientId: k.UserId,
		Amount:      number.FromString("1000"),
		TraceId:     getSettlementId(k.UserId, "LOCKED|OOO|BONUS"),
		Memo:        "Locked OOO Bonus",
	}, config.ClientId, config.SessionId, config.SessionKey, config.SessionAssetPIN, config.PinToken)
	if err != nil {
		return session.ServerError(ctx, err)
	}

	input := &bot.TransferInput{
		AssetId:     config.OOOAssetId,
		RecipientId: config.EngineUserId,
		Amount:      number.FromString("0.00000001"),
		TraceId:     getSettlementId(k.UserId, "USER|SIG|REGISTER"),
		Memo:        base64.StdEncoding.EncodeToString(sig),
	}
	err = bot.CreateTransfer(ctx, input, k.UserId, k.SessionId, k.SessionKey, k.PlainPIN, k.PinToken)
	if err != nil {
		return session.ServerError(ctx, err)
	}

	k.OceanKey = hex.EncodeToString(oceanKey)
	return nil
}

func getSettlementId(id, modifier string) string {
	h := md5.New()
	io.WriteString(h, id)
	io.WriteString(h, modifier)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}

func (k *PoolKey) setupPIN(ctx context.Context) error {
	pin, err := generateSixDigitCode(ctx)
	if err != nil {
		return session.ServerError(ctx, err)
	}
	encryptedPIN, err := bot.EncryptPIN(ctx, pin, k.PinToken, k.SessionId, k.SessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return session.ServerError(ctx, err)
	}
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
	k.PlainPIN = pin
	return nil
}

func poolKeyFromRow(row *spanner.Row) (*PoolKey, error) {
	var k PoolKey
	err := row.Columns(&k.UserId, &k.SessionId, &k.SessionKey, &k.PinToken, &k.EncryptedPIN, &k.EncryptionHeader, &k.OceanKey, &k.CreatedAt)
	return &k, err
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
