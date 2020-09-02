package main

import (
	"context"
	"flag"
	"log"

	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/MixinNetwork/ocean.one/example/durable"
	"github.com/MixinNetwork/ocean.one/example/services"
)

func main() {
	service := flag.String("service", "http", "run a service")
	flag.Parse()

	spanner, err := durable.OpenSpannerClient(context.Background(), config.GoogleCloudSpanner)
	if err != nil {
		log.Panicln(err)
	}
	defer spanner.Close()

	switch *service {
	case "http":
		err := StartServer(spanner)
		if err != nil {
			log.Println(err)
		}
	default:
		hub := services.NewHub(spanner)
		err := hub.StartService(*service)
		if err != nil {
			log.Println(err)
		}
	}
}
