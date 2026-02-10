package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/knightfall22/Phylax/publisher"
	"github.com/knightfall22/Phylax/simulation/config"
)

// Sensor reading to be sent to NATS
type SensorReading struct {
	SensorID   string `json:"sensor_id"`
	SensorZone string `json:"sensor_zone"`
	//Unix timestamp
	Timestamp    int64   `json:"timestamp"`
	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	CO           float64 `json:"co"`
	BatteryLevel float64 `json:"battery_level"`
}

// TODO: Add proper logger
func (sr *SensorReading) logReading() {
	log.Printf("Sensor(%q) Temp: %.2f, Humidity: %.2f, CO: %.2f, BatteryLevel: %.2f, Zone: %q, Time: %d\n",
		sr.SensorID, sr.Temperature, sr.Humidity, sr.CO, sr.BatteryLevel, sr.SensorZone, sr.Timestamp,
	)
}

// Maintains the state of a single sensor
type SensorState struct {
	ID     string
	ZoneID string

	// GLOBAL: Pointer to the hot-reloadable config (Chaos/Events)
	GlobalConfig *config.SimulationConfig

	// LOCAL: Specific physics for this sensor instance
	// The "Thermostat" (Immutable target for this specific sensor)
	TargetTemp     float64
	TargetHumidity float64

	// The "Thermometer" (Mutable current physics)
	// Initial values start at the target (plus/minus some noise)
	Temperature float64
	Humidity    float64

	CO           float64
	BatteryLevel float64

	isOnFire bool
}

type SensorOptions struct {
	ZoneID          string
	ZoneTemperature float64
	ZoneHumidity    float64
	ID              string
	Cfg             *config.SimulationConfig
}

// NewSensor initializes a sensor with a starting state
func NewSensor(opts SensorOptions) *SensorState {
	return &SensorState{
		ID:             opts.ID,
		ZoneID:         opts.ZoneID,
		GlobalConfig:   opts.Cfg,
		TargetTemp:     opts.ZoneTemperature,
		TargetHumidity: opts.ZoneHumidity,
		Temperature:    opts.ZoneTemperature,
		Humidity:       opts.ZoneHumidity,
		CO:             0.0,
		BatteryLevel:   100.0,
		isOnFire:       false,
	}
}

// Advances the physics value by one step in time using Markov Chain
func (s *SensorState) Tick() *SensorReading {
	// Determine if the global disaster settings apply to THIS sensor.
	// Matches if config says "All" OR if config matches my specific ZoneID.
	// If DisasterZone is "None", this is always false.
	isZoneAffected := s.GlobalConfig.DisasterZone == "All" || s.GlobalConfig.DisasterZone == s.ZoneID

	//Simualte Network fault
	if rand.Float64() < s.GlobalConfig.PacketLossRate {
		return nil //packet dropped
	}

	//Battery physics(linear decay)
	s.BatteryLevel -= s.GlobalConfig.BatteryDrain
	if s.BatteryLevel < 0 {
		s.BatteryLevel = 0
		return nil //battery dead
	}

	//CO physics(disaster logic)
	// We only roll the dice for fire IF we are in the affected zone.
	// This ensures the "Kitchen" doesn't catch fire just because the "ServerRoom" config is high.
	if !s.isOnFire && isZoneAffected && rand.Float64() < s.GlobalConfig.FireProbability {
		s.isOnFire = true
	}

	if s.isOnFire {
		//Extinguish fire
		if !isZoneAffected || s.GlobalConfig.FireProbability == 0 {
			s.isOnFire = false
		}
		// CO
		// Rapid spike. SpikeRate determines the magnitude/severity.
		// e.g. 10.0 + random(50.0)
		s.CO += 10.0 + rand.Float64()*s.GlobalConfig.SpikeRate
		if s.CO >= 1000 {
			s.CO = 1000
		}

		// B. Temperature (Thermal Runaway)
		// Fire adds heat regardless of HVAC.
		// We do NOT use TargetTemp here; fire ignores the thermostat.
		s.Temperature += 1.5 + (rand.Float64() * 0.5)

		// C. Humidity (Drying Effect)
		s.Humidity -= 0.5

	} else {
		// Carbon Monoxide (Decay)
		// Clears out slowly if fire stops
		s.CO = math.Max(0, s.CO-1.0)

		// B. Temperature (HVAC / Mean Reversion)
		// Pull current Temp towards s.TargetTemp
		diff := s.TargetTemp - s.Temperature

		// Add Noise (Global fluctuation setting)
		noise := (rand.Float64() - 0.5) * s.GlobalConfig.TempFluctuation

		// Apply Physics: Move 10% of the way to target + noise
		s.Temperature += (diff * 0.1) + noise

		// Humidity (Mean Reversion)
		// Pull towards s.TargetHumidity
		humDiff := s.TargetHumidity - s.Humidity
		humDrift := (rand.Float64() - 0.5) * 2.0
		s.Humidity += (humDiff * 0.05) + humDrift
	}

	// Clamp to realistic 0-100%
	if s.Humidity > 100 {
		s.Humidity = 100
	}
	if s.Humidity < 0 {
		s.Humidity = 0
	}

	// Return the View (DTO)
	return &SensorReading{
		SensorID:     s.ID,
		SensorZone:   s.ZoneID,
		Timestamp:    time.Now().UTC().UnixMilli(),
		Temperature:  round(s.Temperature),
		Humidity:     round(s.Humidity),
		CO:           round(s.CO),
		BatteryLevel: round(s.BatteryLevel),
	}
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

func StartSimulator(
	ctx context.Context,
	sensor *SensorState,
	cfg *config.SimulationConfig,
	publisher publisher.Publisher,
	onError func(err error),
	wg *sync.WaitGroup,
) {

	//Prevent thurdering herd issue
	time.Sleep(time.Duration(rand.Int63n(int64(cfg.TickRate))))

	ticker := time.NewTicker(cfg.TickRate)
	defer func() {
		ticker.Stop()
		fmt.Println(sensor.ID)
		wg.Done()
	}()

	topic := fmt.Sprintf("sensors.%s.%s", sensor.ZoneID, sensor.ID)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data := sensor.Tick()

			if data != nil {
				byt, err := json.Marshal(data)
				if err != nil {
					onError(err)
				}
				err = publisher.Publish(topic, byt)
				if err != nil {
					onError(err)
				}
				data.logReading()

			}
		}
	}
}
