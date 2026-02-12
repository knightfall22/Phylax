package main

import (
	"context"
	"log"
	"net/http"

	"github.com/knightfall22/Phylax/config"
	"github.com/knightfall22/Phylax/internals/processor"
	"github.com/knightfall22/Phylax/publisher"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type App struct {
	Processor   *processor.Processor
	Publisher   *publisher.NatsPublisher
	consumerCtx jetstream.ConsumeContext
}

func Run(ctx context.Context, conf *config.Config) *App {
	processor := processor.NewProcessor(ctx)
	processor.Start(ctx)

	nc, err := publisher.NATSConnect(ctx, publisher.NATSConnectionOptions{
		TLSEnabled: conf.TLSEnabled,
		ClientCert: conf.ClientCert,
		ClientKey:  conf.ClientKey,
		RootCA:     conf.RootCA,
		URL:        conf.NATSURL,
	})
	if err != nil {
		log.Panicf("[Error] cannot connect NATS server %v\n", err)
	}

	consumerCtx, err := nc.Consume(ctx, func(m jetstream.Msg) {
		processor.Submit(m)

	})
	if err != nil {
		log.Panicf("[Error] cannot connect NATS server %v\n", err)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Prometheus metrics available at :2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Printf("Metrics server failed: %v", err)
		}
	}()

	return &App{
		Processor:   processor,
		Publisher:   nc,
		consumerCtx: consumerCtx,
	}
}

func (a *App) Close() {
	a.Publisher.Close()
	a.consumerCtx.Drain()
	a.consumerCtx.Stop()
}
