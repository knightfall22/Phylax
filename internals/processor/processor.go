package processor

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/knightfall22/Phylax/api/v1"
	"github.com/knightfall22/Phylax/internals/metrics"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

const (
	BatchSize    = 1500
	FlusInterval = 1
)

var workerCount = runtime.NumCPU()

type batchItem struct {
	data *pb.SensorReading
	msg  jetstream.Msg
}
type Processor struct {
	input  chan jetstream.Msg
	dbPool *pgxpool.Pool
}

func NewProcessor(ctx context.Context, dbstring string) *Processor {
	config, err := pgxpool.ParseConfig(dbstring)
	if err != nil {
		log.Fatal("Unable to parse DB config:", err)
	}

	config.MaxConns = int32(workerCount)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("Unable to connect to DB:", err)
	}
	return &Processor{
		input:  make(chan jetstream.Msg, 50000),
		dbPool: pool,
	}
}

func (p *Processor) Start(ctx context.Context) {
	for i := range workerCount {
		go p.workerLoop(ctx, i)
	}
}

// Submits reading to queue
func (p *Processor) Submit(data jetstream.Msg) {
	p.input <- data
}

func (p *Processor) flushBatch(ctx context.Context, batch []*batchItem) {
	if len(batch) == 0 {
		return
	}

	rows := [][]interface{}{}
	for _, reading := range batch {
		rows = append(rows, []interface{}{
			reading.data.Timestamp,
			reading.data.SensorId,
			reading.data.SensorZone,
			reading.data.Temperature,
			reading.data.Humidity,
			reading.data.CoLevel,
			reading.data.BatteryLevel,
		})
	}

	_, err := p.dbPool.CopyFrom(
		ctx,
		pgx.Identifier{"sensor_readings"},
		[]string{"time", "sensor_id", "zone", "temperature", "humidity", "co_level", "battery_level"},
		pgx.CopyFromRows(rows),
	)

	//Message is not acknowledged when error exists.
	//This forces the NATS server to retry the message
	if err != nil {
		log.Printf("ERROR: Failed to flush batch to DB: %v", err)
		return
	}

	for _, item := range batch {
		item.msg.Ack()
	}
}

// Core of the processor. Fans in all readings from NATS.
// Batches all readings in-memory then flush when interval elapses or the batch is full
func (p *Processor) workerLoop(ctx context.Context, i int) {
	batch := make([]*batchItem, 0, BatchSize)
	ticker := time.NewTicker(FlusInterval * time.Second)
	defer ticker.Stop()

	timeSince := time.Now()
	for {
		select {
		case rawMsg := <-p.input:
			var reading pb.SensorReading
			if err := proto.Unmarshal(rawMsg.Data(), &reading); err != nil {
				log.Printf("Invalid Protobuf: %v", err)
				// If it's garbage, Ack it to remove from queue
				rawMsg.Ack()
				continue
			}

			batch = append(batch, &batchItem{data: &reading, msg: rawMsg})
			metrics.SensorReadings.WithLabelValues(reading.SensorZone).Inc()
			metrics.SetReadingsGauge(&reading)

			if len(batch) >= BatchSize {
				p.flushBatch(ctx, batch)
				fmt.Printf("Worker %d: Batch Full! Flushing %d took: %s\n", i, len(batch), time.Since(timeSince))
				metrics.BatchSize.Observe(float64(len(batch)))
				//Reset batch buffer
				timeSince = time.Now()
				batch = batch[:0]
				ticker.Reset(FlusInterval * time.Second)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				p.flushBatch(ctx, batch)
				fmt.Printf("Worker %d: Flushed: %d\n", i, len(batch))
				//Reset batch buffer
				metrics.BatchSize.Observe(float64(len(batch)))
				timeSince = time.Now()
				batch = batch[:0]
			}

		case <-ctx.Done():
			if len(batch) > 0 {
				p.flushBatch(ctx, batch)
			}
		}
	}
}
