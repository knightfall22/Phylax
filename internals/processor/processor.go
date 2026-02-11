package processor

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	pb "github.com/knightfall22/Phylax/api/v1"
	"google.golang.org/protobuf/proto"
)

const (
	BatchSize    = 1000
	FlusInterval = 1
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
	batch := make([]*pb.SensorReading, 0, BatchSize)
	ticker := time.NewTicker(FlusInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case rawMsg := <-p.input:
			var reading pb.SensorReading
			if err := proto.Unmarshal(rawMsg, &reading); err != nil {
				log.Printf("Invalid Protobuf: %v", err)
				continue
			}
			batch = append(batch, &reading)

			if len(batch) >= BatchSize {
				p.flushBatch(ctx)
				//Reset batch buffer
				batch = batch[:0]
			}

		case <-ticker.C:
			fmt.Printf("Worker %d: Flushed: %d\n", i, len(batch))
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
