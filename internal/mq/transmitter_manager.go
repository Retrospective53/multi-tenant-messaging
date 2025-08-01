package mq

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TransmitterManager struct {
	mu           sync.RWMutex
	transmitters map[uuid.UUID]*transmitter
	wg           sync.WaitGroup // Track all workers
}

type transmitter struct {
	mu          sync.RWMutex
	taskChan    chan []byte
	cancel      context.CancelFunc
	workerCount int
}

func NewTransmitterManager() *TransmitterManager {
	return &TransmitterManager{
		transmitters: make(map[uuid.UUID]*transmitter),
	}
}

// Start a transmitter with initial worker count
func (tm *TransmitterManager) StartTransmitter(tenantID uuid.UUID, workers int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.transmitters[tenantID]; exists {
		log.Printf("[Transmitter] Tenant %s already exists", tenantID)
		return errors.New("transmitter already exists")
	}

	ctx, cancel := context.WithCancel(context.Background())
	taskChan := make(chan []byte, 100)

	t := &transmitter{
		taskChan:    taskChan,
		cancel:      cancel,
		workerCount: workers,
	}
	tm.transmitters[tenantID] = t

	for i := 0; i < workers; i++ {
		tm.wg.Add(1)
		go tm.worker(ctx, tenantID, taskChan, i)
	}

	log.Printf("[Transmitter] Started transmitter for tenant %s with %d workers", tenantID, workers)
	return nil
}

// Enqueue message
func (tm *TransmitterManager) Enqueue(tenantID uuid.UUID, msg []byte) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	t, ok := tm.transmitters[tenantID]
	if !ok {
		log.Printf("[Transmitter] No transmitter found for tenant %s", tenantID)
		return
	}

	select {
	case t.taskChan <- msg:
	default:
		log.Printf("[Transmitter] Task queue full for tenant %s, dropping message", tenantID)
	}
}

// Update worker count dynamically
func (tm *TransmitterManager) UpdateWorkerCount(tenantID uuid.UUID, newCount int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	t, ok := tm.transmitters[tenantID]
	if !ok {
		log.Printf("[Transmitter] No transmitter found for tenant %s", tenantID)
		return
	}

	t.cancel() // stop old workers

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.taskChan = make(chan []byte, 100)
	t.workerCount = newCount

	for i := 0; i < newCount; i++ {
		tm.wg.Add(1)
		go tm.worker(ctx, tenantID, t.taskChan, i)
	}

	log.Printf("[Transmitter] Updated worker count to %d for tenant %s", newCount, tenantID)
}

// Stop transmitter for one tenant
func (tm *TransmitterManager) StopTransmitter(tenantID uuid.UUID) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	t, ok := tm.transmitters[tenantID]
	if !ok {
		log.Printf("[Transmitter] No transmitter found for tenant %s", tenantID)
		return
	}

	t.cancel()
	delete(tm.transmitters, tenantID)

	log.Printf("[Transmitter] Stopped transmitter for tenant %s", tenantID)
}

// Worker function
func (tm *TransmitterManager) worker(ctx context.Context, tenantID uuid.UUID, tasks <-chan []byte, id int) {
	defer tm.wg.Done()
	log.Printf("[Transmitter] Worker %d started for tenant %s", id, tenantID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Transmitter] Worker %d exiting for tenant %s", id, tenantID)
			return
		case msg := <-tasks:
			// Simulate work
			log.Printf("[Transmitter] Tenant %s | Worker %d processing: %s", tenantID, id, string(msg))
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Shutdown all transmitters gracefully
func (tm *TransmitterManager) ShutdownAndWait() {
	log.Println("[Transmitter] Shutting down all transmitters...")

	tm.mu.Lock()
	for tenantID, t := range tm.transmitters {
		log.Printf("[Transmitter] Stopping tenant %s", tenantID)
		t.cancel()
	}
	tm.transmitters = make(map[uuid.UUID]*transmitter)
	tm.mu.Unlock()

	tm.wg.Wait()
	log.Println("[Transmitter] All workers stopped")
}
