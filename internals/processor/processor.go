package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/knightfall22/Phylax/internals/types"
)

const (
	BatchSize    = 1000
	FlusInterval = 5
)

var WorkerCount = runtime.NumCPU()

type Processor struct {
	input chan []byte
}

func NewProcessor() *Processor {
	return &Processor{
		input: make(chan []byte, 5000),
	}
}

func (p *Processor) Start(ctx context.Context) {
	for i := range WorkerCount {
		go p.workerLoop(ctx, i)
	}
}

// Submits reading to queue
func (p *Processor) Submit(data []byte) {
	p.input <- data
}

func (p *Processor) flushBatch(ctx context.Context) {
	// log.Println("Flushing....")
}

// Core of the processor. Fans in all readings from NATS.
// Batches all reading in-memory then flush when interval elapses or the batch is full
func (p *Processor) workerLoop(ctx context.Context, i int) {
	batch := make([]*types.SensorReading, 0, BatchSize)
	ticker := time.NewTicker(FlusInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-p.input:
			var reading types.SensorReading
			json.Unmarshal(data, &reading)
			batch = append(batch, &reading)

			if len(batch) >= BatchSize {
				p.flushBatch(ctx)
				fmt.Printf("Worker %d: Flushed: %d\n", i, len(batch))
				//Reset batch buffer
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				p.flushBatch(ctx)
				//Reset batch buffer
				batch = batch[:0]
			}

		case <-ctx.Done():
			if len(batch) > 0 {
				p.flushBatch(ctx)
			}
		}
	}
}
