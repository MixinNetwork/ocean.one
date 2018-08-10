package persistence

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/ocean.one/config"
	"google.golang.org/api/iterator"
)

const encryptionHeaderLength = 16

type Broker struct {
	BrokerId         string    `spanner:"broker_id"`
	SessionId        string    `spanner:"session_id"`
	SessionKey       string    `spanner:"session_key"`
	PINToken         string    `spanner:"pin_token"`
	EncryptedPIN     string    `spanner:"encrypted_pin"`
	EncryptionHeader []byte    `spanner:"encryption_header"`
	CreatedAt        time.Time `spanner:"created_at"`

	DecryptedPIN string
}

func AllBrokers(ctx context.Context, decryptPIN bool) ([]*Broker, error) {
	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{SQL: "SELECT * FROM brokers"})
	defer it.Stop()

	brokers := []*Broker{
		&Broker{
			BrokerId:     config.ClientId,
			SessionId:    config.SessionId,
			SessionKey:   config.SessionKey,
			PINToken:     config.PinToken,
			DecryptedPIN: config.SessionAssetPIN,
		},
	}

	for {
		row, err := it.Next()
		if err == iterator.Done {
			return brokers, nil
		} else if err != nil {
			return brokers, err
		}
		var broker Broker
		err = row.ToStruct(&broker)
		if err != nil {
			return brokers, err
		}
		if decryptPIN {
			err = broker.decryptPIN()
			if err != nil {
				return brokers, err
			}
		}
		brokers = append(brokers, &broker)
	}
}

func AddBroker(ctx context.Context) (*Broker, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return nil, err
	}
	sessionSecret := base64.StdEncoding.EncodeToString(publicKeyBytes)

	data, err := json.Marshal(map[string]string{
		"session_secret": sessionSecret,
		"full_name":      fmt.Sprintf("Ocean %x", md5.Sum(publicKeyBytes)),
	})
	if err != nil {
		return nil, err
	}
	token, err := bot.SignAuthenticationToken(config.ClientId, config.SessionId, config.SessionKey, "POST", "/users", string(data))
	if err != nil {
		return nil, err
	}

	body, err := bot.Request(ctx, "POST", "/users", data, token)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}

	broker := &Broker{
		BrokerId:  resp.Data.UserId,
		SessionId: resp.Data.SessionId,
		SessionKey: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})),
		PINToken:  resp.Data.PinToken,
		CreatedAt: time.Now(),
	}

	err = broker.setupPIN(ctx)
	if err != nil {
		return nil, err
	}
	insertBroker, err := spanner.InsertStruct("brokers", broker)
	if err != nil {
		return nil, err
	}
	_, err = Spanner(ctx).Apply(ctx, []*spanner.Mutation{insertBroker})
	return broker, err
}

func (b *Broker) decryptPIN() error {
	privateBlock, _ := pem.Decode([]byte(config.AssetPrivateKey))
	privateKey, err := x509.ParsePKCS1PrivateKey(privateBlock.Bytes)
	if err != nil {
		return err
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, b.EncryptionHeader[encryptionHeaderLength:], nil)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return err
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(b.EncryptedPIN)
	if err != nil {
		return err
	}
	iv := cipherBytes[:aes.BlockSize]
	source := cipherBytes[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(source, source)

	length := len(source)
	unpadding := int(source[length-1])
	b.DecryptedPIN = string(source[:length-unpadding])
	return nil
}

func (b *Broker) setupPIN(ctx context.Context) error {
	pin, err := generateSixDigitCode(ctx)
	if err != nil {
		return err
	}
	encryptedPIN, err := bot.EncryptPIN(ctx, pin, b.PINToken, b.SessionId, b.SessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return err
	}
	data, _ := json.Marshal(map[string]string{"pin": encryptedPIN})

	token, err := bot.SignAuthenticationToken(b.BrokerId, b.SessionId, b.SessionKey, "POST", "/pin/update", string(data))
	if err != nil {
		return err
	}
	body, err := bot.Request(ctx, "POST", "/pin/update", data, token)
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
		return resp.Error
	}
	encryptedPIN, encryptionHeader, err := encryptPIN(ctx, pin)
	if err != nil {
		return err
	}
	b.EncryptedPIN = encryptedPIN
	b.EncryptionHeader = encryptionHeader
	return nil
}

func encryptPIN(ctx context.Context, pin string) (string, []byte, error) {
	aesKey := make([]byte, 32)
	_, err := rand.Read(aesKey)
	if err != nil {
		return "", nil, err
	}
	publicBytes, err := base64.StdEncoding.DecodeString(config.AssetPublicKey)
	if err != nil {
		return "", nil, err
	}
	assetPublicKey, err := x509.ParsePKCS1PublicKey(publicBytes)
	if err != nil {
		return "", nil, err
	}
	aesKeyEncrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, assetPublicKey, aesKey, nil)
	if err != nil {
		return "", nil, err
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
		return "", nil, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherBytes[aes.BlockSize:], plainBytes)
	return base64.StdEncoding.EncodeToString(cipherBytes), encryptionHeader, nil
}

func generateSixDigitCode(ctx context.Context) (string, error) {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	c := binary.LittleEndian.Uint64(b[:]) % 1000000
	if c < 100000 {
		c = 100000 + c
	}
	return fmt.Sprint(c), nil
}
