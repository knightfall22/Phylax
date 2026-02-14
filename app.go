package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/knightfall22/Phylax/config"
	"github.com/knightfall22/Phylax/internals/processor"
	"github.com/knightfall22/Phylax/publisher"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/knightfall22/Phylax/internals/metrics"
)

//go:embed db/migration/*.sql
var embedMigrations embed.FS

type App struct {
	Processor   *processor.Processor
	Publisher   *publisher.NatsPublisher
	consumerCtx jetstream.ConsumeContext
}

func Run(ctx context.Context, conf *config.Config) *App {

	connectionStream := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		conf.DBUser, conf.DBPassword, conf.DBHost, conf.DBPort, conf.DBName)

	db, err := sql.Open("pgx", connectionStream)
	if err != nil {
		log.Fatalf("Failed to open DB for migrations: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set goose dialect: %v", err)
	}

	if err := goose.Up(db, "db/migration"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	processor := processor.NewProcessor(ctx, connectionStream)
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
