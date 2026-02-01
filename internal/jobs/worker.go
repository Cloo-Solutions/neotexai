package jobs

import (
	"context"
	"log"
	"time"
)

// JobProcessor defines the interface for processing jobs
type JobProcessor interface {
	ProcessJobs(ctx context.Context) error
}

// Worker represents a background job worker
type Worker struct {
	processor    JobProcessor
	pollInterval time.Duration
	stopChan     chan struct{}
	doneChan     chan struct{}
}

// NewWorker creates a new Worker instance
func NewWorker(processor JobProcessor, pollInterval time.Duration) *Worker {
	return &Worker{
		processor:    processor,
		pollInterval: pollInterval,
		stopChan:     make(chan struct{}),
		doneChan:     make(chan struct{}),
	}
}

// Start begins the worker's polling loop
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	defer close(w.doneChan)

	log.Printf("Worker started with poll interval: %v", w.pollInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker stopped: context cancelled")
			return
		case <-w.stopChan:
			log.Println("Worker stopped: stop signal received")
			return
		case <-ticker.C:
			if err := w.processor.ProcessJobs(ctx); err != nil {
				log.Printf("Error processing jobs: %v", err)
			}
		}
	}
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	close(w.stopChan)
	<-w.doneChan
	log.Println("Worker shutdown complete")
}
