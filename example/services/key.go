package services

import (
	"context"
	"log"
	"time"

	"github.com/MixinNetwork/ocean.one/example/models"
)

type KeyService struct{}

func (service *KeyService) Healthy(ctx context.Context) bool {
	if err := standardServiceHealth(ctx); err != nil {
		return false
	}
	return true
}

func (service *KeyService) Run(ctx context.Context) error {
	if err := standardServiceHealth(ctx); err != nil {
		return err
	}
	for {
		key, err := models.GeneratePoolKey(ctx)
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
		}
		log.Println(key)
	}
}
