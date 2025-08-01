// internal/mq/consumer_manager.go
package mq

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/retrospective53/multi-tenant/internal/service"
	"github.com/streadway/amqp"
)

type ConsumerManager struct {
	conn           *amqp.Connection
	service        *service.MessageService
	consumers      map[uuid.UUID]context.CancelFunc
	mu             sync.Mutex
	transmitterMgr *TransmitterManager
}

func NewConsumerManager(conn *amqp.Connection, svc *service.MessageService, transmitterMgr *TransmitterManager) *ConsumerManager {
	return &ConsumerManager{
		conn:           conn,
		service:        svc,
		consumers:      make(map[uuid.UUID]context.CancelFunc),
		transmitterMgr: transmitterMgr,
	}
}

// StartConsumer creates a consumer for a specific tenant
func (cm *ConsumerManager) StartConsumer(tenantID uuid.UUID) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.consumers[tenantID]; exists {
		log.Printf("Consumer for tenant %s already running", tenantID)
		return nil
	}

	channel, err := cm.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	queueName := fmt.Sprintf("tenant_%s", tenantID.String())
	_, err = channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Start the transmitter with a default worker count, e.g., 5
	err = cm.transmitterMgr.StartTransmitter(tenantID, 5)
	if err != nil {
		return fmt.Errorf("failed to start transmitter: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cm.consumers[tenantID] = cancel

	go func() {
		defer channel.Close()
		msgs, err := channel.Consume(queueName, "", true, false, false, false, nil)
		if err != nil {
			log.Printf("Failed to consume queue %s: %v", queueName, err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				log.Printf("Stopping consumer for tenant %s", tenantID)
				cm.transmitterMgr.StopTransmitter(tenantID)
				return
			case d := <-msgs:
				if len(d.Body) == 0 {
					continue
				}
				// Hand off to transmitter, not directly saving
				cm.transmitterMgr.Enqueue(tenantID, d.Body)
			}
		}
	}()

	log.Printf("Started consumer and transmitter for tenant %s", tenantID)
	return nil
}

func (cm *ConsumerManager) StopConsumer(tenantID uuid.UUID) error {
	cm.mu.Lock()
	cancel, exists := cm.consumers[tenantID]
	if !exists {
		cm.mu.Unlock()
		log.Printf("No consumer found for tenant %s", tenantID)
		return nil // or return an error if you want to enforce it
	}
	delete(cm.consumers, tenantID) // remove from map
	cm.mu.Unlock()

	// Send cancellation signal to stop goroutine
	cancel()

	// Open a new channel just to delete the queue
	channel, err := cm.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel for queue deletion: %w", err)
	}
	defer channel.Close()

	queueName := fmt.Sprintf("tenant_%s", tenantID.String())
	_, err = channel.QueueDelete(queueName, false, false, false)
	if err != nil {
		return fmt.Errorf("failed to delete queue: %w", err)
	}

	log.Printf("Stopped and cleaned up consumer for tenant %s", tenantID)
	return nil
}

func (cm *ConsumerManager) UpdateConcurrency(tenantID uuid.UUID, newCount int) error {
	cm.transmitterMgr.UpdateWorkerCount(tenantID, newCount)
	return nil
}

func (cm *ConsumerManager) StopAllConsumers() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for tenantID, cancel := range cm.consumers {
		log.Printf("Stopping consumer for tenant %s", tenantID)
		cancel() // cancels the context
		delete(cm.consumers, tenantID)
	}
}
