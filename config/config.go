package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	TLSEnabled bool   `mapstructure:"tls_enabled"`
	ClientCert string `mapstructure:"client_cert"`
	ClientKey  string `mapstructure:"client_key"`
	RootCA     string `mapstructure:"root_ca"`

	NATSURL string `mapstructure:"nats_url"`
}

func LoadConfigurations() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Panicf("Faild to read configuration. %v", err)
	}

	var config Config
	viper.Unmarshal(&config)

	return &config
}
