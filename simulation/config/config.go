package config

import (
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type ZoneConfig struct {
	Name     string  `mapstructure:"name"`
	Percent  float64 `mapstructure:"percent"` // 0.0 to 1.0
	BaseTemp float64 `mapstructure:"base_temp"`
	BaseHum  float64 `mapstructure:"base_hum"`
}

// Configuration of simulation.
type SimulationConfig struct {
	//Chaos Parameters (0.0 to 1.0)
	TempFluctuation float64 `mapstructure:"temp_fluctuation"` // How much temp wobbles naturally (Noise)
	PacketLossRate  float64 `mapstructure:"packet_loss_rate"` //Chances that a packet is lost
	SpikeRate       float64 `mapstructure:"spike_rate"`       //CO Spike rate during fire
	FireProbability float64 `mapstructure:"fire_probability"` //Chances that a fire will ignite
	BatteryDrain    float64 `mapstructure:"battery_drain"`
	DisasterZone    string  `mapstructure:"disaster_zone"`

	// The List of Zones
	Zones       []ZoneConfig `mapstructure:"zones"`
	DefaultZone ZoneConfig   `mapstructure:"default_zone"`

	TickRate    time.Duration `mapstructure:"tick_rate"`
	SensorCount int           `mapstructure:"sensor_count"`

	TLSEnabled bool   `mapstructure:"tls_enabled"`
	ClientCert string `mapstructure:"client_cert"`
	ClientKey  string `mapstructure:"client_key"`
	RootCA     string `mapstructure:"root_ca"`

	NATSURL string `mapstructure:"nats_url"`
}

type EditableConfig struct {
	Config SimulationConfig
	mu     sync.Mutex
}

func LoadConguration() *EditableConfig {
	viper.SetConfigName("simulation-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/config")
	viper.AddConfigPath(".")

	var cfg EditableConfig

	if err := viper.ReadInConfig(); err != nil {
		log.Panicf("Faild to read configuration. %v", err)
	}

	cfg.mu.Lock()
	viper.Unmarshal(&cfg.Config)
	cfg.mu.Unlock()

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)

		cfg.mu.Lock()
		defer cfg.mu.Unlock()

		viper.Unmarshal(&cfg.Config)
	})
	viper.WatchConfig()

	return &cfg
}
