package metrics

import (
	"time"

	pb "github.com/knightfall22/Phylax/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Total readings processed, labeled by zone
var SensorReadings = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "phylax_sensor_readings_total",
		Help: "The total number of processed sensor readings",
	},
	[]string{"zone"}, // This allows per-zone filtering in Grafana
)

// Histogram of batch sizes
var BatchSize = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "phylax_batch_flush_size",
		Help:    "Size of the batches being flushed to Postgres",
		Buckets: []float64{100, 500, 1000, 1500, 2000},
	},
)

var TempHistogram = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "phylax_temp_distribution",
		Help: "Distribution of temperature readings",
		// Specific buckets for "Disaster" detection
		Buckets: []float64{0, 20, 30, 50, 100, 300},
	},
	[]string{"zone"},
)

var COHistogram = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "phylax_co_distribution",
		Help: "Distribution of CO readings (PPM)",
		// Buckets: 0-9 (Safe), 10-30 (Caution), >50 (Danger)
		Buckets: []float64{0, 9, 30, 50, 100, 400},
	},
	[]string{"zone"},
)

var HumidityHistogram = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "phylax_humidity_distribution_percent",
		Help: "Distribution of humidity readings (Percent)",
		// Buckets: 0-20 (Dry), 40-60 (Comfort), >80 (Wet), >90 (Danger)
		Buckets: []float64{10, 20, 40, 60, 80, 90, 100},
	},
	[]string{"zone"},
)

var BatteryHistogram = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "phylax_battery_distribution",
		Help:    "Distribution of battery levels (Percent)",
		Buckets: []float64{10, 20, 30, 50, 80, 100},
	},
	[]string{"zone"},
)

var DataLag = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "phylax_data_lag_seconds",
		Help:    "Time difference between sensor timestamp and processing time",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10}, // Focus on sub-second speed
	},
	[]string{"zone"},
)

func SetReadingsGauge(reading *pb.SensorReading) {
	TempHistogram.WithLabelValues(reading.SensorZone).Observe(reading.Temperature)
	COHistogram.WithLabelValues(reading.SensorZone).Observe(reading.CoLevel)
	HumidityHistogram.WithLabelValues(reading.SensorZone).Observe(reading.Humidity)
	BatteryHistogram.WithLabelValues(reading.SensorZone).Observe(reading.BatteryLevel)

	creationTime := time.UnixMilli(reading.Timestamp)
	lag := time.Since(creationTime).Seconds()
	DataLag.WithLabelValues(reading.SensorZone).Observe(lag)
}
