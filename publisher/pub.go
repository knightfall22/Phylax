package publisher

import (
	globalConfig "github.com/knightfall22/Phylax/config"
	"github.com/knightfall22/Phylax/simulation/config"
	"github.com/nats-io/nats.go"
)

type Publisher interface {
	Publish(subject string, payload []byte) error
}

type NatsPublisher struct {
	nc *nats.Conn
}

func NATSConnect(cfg *config.SimulationConfig) (*NatsPublisher, error) {
	if cfg.NATSURL == "" {
		cfg.NATSURL = nats.DefaultURL
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
	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		return nil, err
	}

	return &NatsPublisher{
		nc: nc,
	}, nil
}

func (p *NatsPublisher) Close() {
	p.nc.Drain()
	p.nc.Close()
}

func (p *NatsPublisher) Publish(subject string, payload []byte) error {
	return p.nc.Publish(subject, payload)
}
