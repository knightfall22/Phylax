package publisher

import (
	"context"
	"time"

	globalConfig "github.com/knightfall22/Phylax/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsPublisher struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	Stream jetstream.Stream
}

type NATSConnectionOptions struct {
	TLSEnabled bool
	ClientCert string
	ClientKey  string
	RootCA     string
	URL        string
}

func NATSConnect(ctx context.Context, cfg NATSConnectionOptions) (*NatsPublisher, error) {
	if cfg.URL == "" {
		cfg.URL = nats.DefaultURL
	}

	var opts []nats.Option
	if cfg.TLSEnabled {
		tlsConfig, err := globalConfig.SetupTLSConfig(globalConfig.TLSConfig{
			CertFile: cfg.ClientCert,
			KeyFile:  cfg.ClientKey,
			CAFile:   cfg.RootCA,
			//Todo: add proper server address
			ServerAddress: "127.0.0.1",
		})
		if err != nil {
			return nil, err
		}

		tls := nats.Secure(tlsConfig)
		opts = append(opts, tls)
	}
	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}

	js, _ := jetstream.New(nc)

	jsConf := jetstream.StreamConfig{
		Name:      "SENSORS_READINGS",
		Retention: jetstream.WorkQueuePolicy,
		Subjects:  []string{"sensors.>"},
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stream, err := js.CreateOrUpdateStream(ctx, jsConf)
	if err != nil {
		return nil, err
	}

	return &NatsPublisher{
		nc:     nc,
		js:     js,
		Stream: stream,
	}, nil
}

func (p *NatsPublisher) Close() {
	p.nc.Drain()
	p.nc.Close()
}

func (p *NatsPublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	_, err := p.js.Publish(ctx, subject, payload)
	return err
}

func (p *NatsPublisher) Consume(ctx context.Context, handler func(jetstream.Msg)) (jetstream.ConsumeContext, error) {
	config := jetstream.ConsumerConfig{
		Name:          "PROCESSOR_WORKERS",
		Durable:       "PROCESSOR_WORKERS",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "sensors.>",
		AckWait:       30 * time.Second,
	}

	consumer, err := p.js.CreateOrUpdateConsumer(ctx, "SENSORS_READINGS", config)
	if err != nil {
		return nil, err
	}

	consumerCxt, err := consumer.Consume(func(msg jetstream.Msg) {
		handler(msg)

		msg.Ack()
	})

	return consumerCxt, err
}
