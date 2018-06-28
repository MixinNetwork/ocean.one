package mixin

import (
	"crypto/cipher"
	"crypto/rsa"
)

type Client struct {
	ClientId  string
	SessionId string
	PinCipher cipher.Block
	Pin       string

	privateKey *rsa.PrivateKey
}

func CreateMixinClient(clientId, sessionId, pinToken, pin string, privateKey *rsa.PrivateKey) *Client {
	client := &Client{
		ClientId:   clientId,
		SessionId:  sessionId,
		Pin:        pin,
		privateKey: privateKey,
	}

	client.loadPinCipher(pinToken)
	return client
}
