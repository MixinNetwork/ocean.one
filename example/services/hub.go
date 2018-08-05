package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/models"
	"github.com/MixinNetwork/ocean.one/example/session"
)

type Hub struct {
	context  context.Context
	services map[string]Service
}

func NewHub(spanner *spanner.Client) *Hub {
	db := durable.WrapDatabase(spanner, nil)
	hub := &Hub{services: make(map[string]Service)}
	hub.context = session.WithDatabase(context.Background(), db)
	hub.registerServices()
	return hub
}

func (hub *Hub) StartService(name string) error {
	service := hub.services[name]
	if service == nil {
		return fmt.Errorf("no service found: %s", name)
	}

	logger, err := durable.NewLoggerClient(config.GoogleCloudProject, config.Environment != "production")
	if err != nil {
		return err
	}
	defer logger.Close()
	ctx := session.WithLogger(hub.context, durable.BuildLogger(logger, name, nil))

	go hub.checkHealth(ctx, name, service)
	return service.Run(ctx)
}

func (hub *Hub) checkHealth(ctx context.Context, name string, service Service) {
	for {
		key := fmt.Sprintf("health-checker-%s", name)
		healthy := fmt.Sprint(service.Healthy(ctx))
		err := models.WriteProperty(ctx, key, healthy)
		if err != nil {
			session.Logger(ctx).Errorf("HUB health checker ERROR %s %s", name, err.Error())
		}
		time.Sleep(3 * time.Second)
	}
}

func (hub *Hub) registerServices() {
	hub.services["key"] = &KeyService{}
	hub.services["candle"] = &CandleService{}
}
