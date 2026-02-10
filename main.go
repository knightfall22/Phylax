package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/knightfall22/Phylax/config"
	"github.com/knightfall22/Phylax/internals/processor"
	"github.com/knightfall22/Phylax/publisher"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	conf := config.LoadConfigurations()
	ctx := context.Background()

	processor := processor.NewProcessor()
	processor.Start(ctx)

	nc, err := publisher.NATSConnect(ctx, publisher.NATSConnectionOptions{
		TLSEnabled: conf.TLSEnabled,
		ClientCert: conf.ClientCert,
		ClientKey:  conf.ClientKey,
		RootCA:     conf.RootCA,
		URL:        conf.NATSURL,
	})
	if err != nil {
		panic(err)
	}

	defer nc.Close()

	consumerCtx, err := nc.Consume(ctx, func(m jetstream.Msg) {
		processor.Submit(m.Data())
	})
	if err != nil {
		panic(err)
	}

	defer func() {
		consumerCtx.Drain()
		consumerCtx.Stop()
	}()

	var sigs chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
}
