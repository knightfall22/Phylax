package types

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
