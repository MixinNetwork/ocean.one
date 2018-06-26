package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinMessenger/ocean.one/config"
	"github.com/MixinMessenger/ocean.one/mixin"
	"github.com/MixinMessenger/ocean.one/persistence"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClientWithConfig(ctx, config.GoogleCloudSpanner, spanner.ClientConfig{NumChannels: 4,
		SessionPoolConfig: spanner.SessionPoolConfig{
			HealthCheckInterval: 5 * time.Second,
		},
	})
	if err != nil {
		log.Panicln(err)
	}

	block, _ := pem.Decode([]byte(config.SessionKey))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Panicln(err)
	}

	mixinClient := mixin.CreateMixinClient(config.ClientId, config.SessionId, config.PinToken, config.SessionAssetPIN, privateKey)
	persist := persistence.CreateSpanner(client)
	NewExchange(persist, mixinClient).Run(ctx)
}
