package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/knightfall22/Phylax/publisher"
	"github.com/knightfall22/Phylax/simulator/config"
)

// Simulator for air quality monitoring sensors.
// The number of sensors and failure rates are configurable.
// Each sensor has a unique ID. And it's data is published to a NATS topic.
// Sensor monitors CO, temperature, humidity, and it's battery level.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.LoadConguration()
	sensorsCounts := cfg.Config.SensorCount

	fmt.Println(cfg.Config.NATSURL)

	natsConn, err := publisher.NATSConnect(ctx, publisher.NATSConnectionOptions{
		TLSEnabled: cfg.Config.TLSEnabled,
		ClientCert: cfg.Config.ClientCert,
		ClientKey:  cfg.Config.ClientKey,
		RootCA:     cfg.Config.RootCA,
		URL:        cfg.Config.NATSURL,
	})
	if err != nil {
		panic(err)
	}

	defer natsConn.Close()

	var wg sync.WaitGroup
	wg.Add(sensorsCounts)

	// Use atomic.Uint64 for thread-safe counting without locks
	var errorCount atomic.Uint64
	maxErrCount := uint64(float64(sensorsCounts) * 0.25)

	errorHandler := func(err error) {
		if err == nil {
			return
		}

		current := errorCount.Add(1)
		log.Printf("[ERROR] %v (Total: %d/%d)", err, current, maxErrCount)
		if current == maxErrCount {
			log.Fatal("[ERROR] Too many errors. Exiting...")
			cancel()
		}
	}

	currentSensorIdx := 0

	for _, zone := range cfg.Config.Zones {
		// Calculate how many sensors belong to this zone
		// e.g. 1000 * 0.05 = 50 sensors
		count := int(zone.Percent * float64(sensorsCounts))

		fmt.Printf("Creating Zone '%s': %d sensors\n", zone.Name, count)
		for range count {
			// Safety check to prevent index out of bounds if config ratios > 1.0
			if currentSensorIdx >= sensorsCounts {
				break
			}

			spawnSensorReaders(ctx, currentSensorIdx, &cfg.Config, zone, errorHandler, natsConn, &wg)
			currentSensorIdx++
		}
	}

	// ASSIGN REMAINDER TO DEFAULT ZONE
	// e.g. Sensors 150 to 999 become "Office"
	remaining := sensorsCounts - currentSensorIdx
	if remaining > 0 {
		fmt.Printf("Creating Default Zone '%s': %d sensors\n", cfg.Config.DefaultZone.Name, remaining)
		for range remaining {
			spawnSensorReaders(
				ctx,
				currentSensorIdx,
				&cfg.Config,
				cfg.Config.DefaultZone,
				errorHandler, natsConn, &wg)

			currentSensorIdx++
		}
	}

	wg.Wait()
}

func spawnSensorReaders(
	ctx context.Context,
	index int,
	cfg *config.SimulationConfig,
	zone config.ZoneConfig,
	onError func(err error),
	publisher *publisher.NatsPublisher,
	wg *sync.WaitGroup,
) {
	id := fmt.Sprintf("sensor-%d", index)

	// Add randomness to the baseline so not every sensor in the room is identical
	// e.g. Server room is 18C, but this specific rack is 18.2C
	randomOffset := (rand.Float64() - 0.5) * 1.0

	sensor := NewSensor(
		SensorOptions{
			ZoneID:          zone.Name,
			ZoneTemperature: zone.BaseTemp + randomOffset,
			ZoneHumidity:    zone.BaseHum + randomOffset,
			ID:              id,
			Cfg:             cfg,
		},
	)

	go StartSimulator(ctx, sensor, cfg, publisher, onError, wg)
}
