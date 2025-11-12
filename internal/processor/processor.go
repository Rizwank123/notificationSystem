package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Rizwank123/notification_system/internal/config"
	"github.com/Rizwank123/notification_system/internal/delivery"
	"github.com/Rizwank123/notification_system/internal/models"
	"github.com/Rizwank123/notification_system/internal/queue"
	"github.com/Rizwank123/notification_system/internal/repository"
	"github.com/Rizwank123/notification_system/internal/strategy"
	"github.com/Rizwank123/notification_system/pkg/logger"
	"github.com/gofrs/uuid/v5"
)

var (
	// Namespace UUID for delivery records
	deliveryNamespace = uuid.NewV5(uuid.NamespaceOID, "delivery-records")
)

type Processor struct {
	cfg              *config.Config
	queue            *queue.Queue
	repo             *repository.NotificationRepository
	batchStrategy    *strategy.BatchStrategy
	priorityStrategy *strategy.PriorityStrategy
	retryStrategy    *strategy.RetryStrategy
	log              logger.Logger
	wg               sync.WaitGroup
	firebaseService  *delivery.FirebaseService
	metrics          *ProcessorMetrics
	mu               sync.Mutex
}

// ProcessorMetrics tracks processor statistics
type ProcessorMetrics struct {
	TotalProcessed     int64
	TotalDelivered     int64
	TotalFailed        int64
	TotalBatched       int64
	TotalRetries       int64
	AverageProcessTime time.Duration
	mu                 sync.RWMutex
}

func NewProcessor(
	cfg *config.Config,
	q *queue.Queue,
	log logger.Logger,
	repo *repository.NotificationRepository,
	firebaseService *delivery.FirebaseService,
) *Processor {
	return &Processor{
		cfg:              cfg,
		queue:            q,
		repo:             repo,
		batchStrategy:    strategy.NewBatchStrategy(cfg),
		priorityStrategy: strategy.NewPriorityStrategy(),
		retryStrategy:    strategy.NewRetryStrategy(cfg),
		log:              log,
		firebaseService:  firebaseService,
		metrics:          &ProcessorMetrics{},
	}
}

// Start starts the notification processor
func (p *Processor) Start(ctx context.Context) error {
	p.log.Info("Starting notification processor", "workers", p.cfg.QueueWorkers)

	priorities := []models.NotificationPriority{
		models.PriorityUrgent,
		models.PriorityHigh,
		models.PriorityMedium,
		models.PriorityLow,
	}

	// Start workers for each priority level
	for _, priority := range priorities {
		workerCount := p.getWorkerCount(priority)
		p.log.Info("Starting workers", "priority", priority, "count", workerCount)

		for i := 0; i < workerCount; i++ {
			p.wg.Add(1)
			go p.worker(ctx, priority, i)
		}
	}

	// Start metrics reporter
	go p.reportMetrics(ctx)

	// Start batch processor
	go p.processBatches(ctx)

	return nil
}

// getWorkerCount returns the number of workers for a priority level
func (p *Processor) getWorkerCount(priority models.NotificationPriority) int {
	switch priority {
	case models.PriorityUrgent:
		return p.cfg.QueueWorkers * 2
	case models.PriorityHigh:
		return p.cfg.QueueWorkers
	case models.PriorityMedium:
		return p.cfg.QueueWorkers / 2
	case models.PriorityLow:
		return p.cfg.QueueWorkers / 4
	default:
		return 1
	}
}

// worker processes notifications from the queue
func (p *Processor) worker(ctx context.Context, priority models.NotificationPriority, workerID int) {
	defer p.wg.Done()

	p.log.Info("Worker started", "priority", priority, "worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			p.log.Info("Worker stopping", "priority", priority, "worker_id", workerID)
			return
		default:
			notification, err := p.queue.Dequeue(ctx, priority)
			if err != nil {
				if err == context.Canceled {
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if notification == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			startTime := time.Now()
			p.processNotification(ctx, notification)
			p.updateMetrics(time.Since(startTime))
		}
	}
}

// processNotification processes a single notification
func (p *Processor) processNotification(ctx context.Context, notification *models.Notification) {
	p.log.Info("Processing notification",
		"id", notification.ID.String(),
		"user_id", notification.UserID.String(),
		"type", notification.Type,
		"priority", notification.Priority,
	)

	// Increment processed counter
	p.metrics.mu.Lock()
	p.metrics.TotalProcessed++
	p.metrics.mu.Unlock()

	// Check if should be batched
	if p.shouldBatch(notification) {
		p.log.Debug("Notification batched", "id", notification.ID.String())
		p.batchStrategy.AddToBatch(notification)

		p.metrics.mu.Lock()
		p.metrics.TotalBatched++
		p.metrics.mu.Unlock()
		return
	}

	// Deliver immediately
	p.deliver(ctx, notification)
}

// shouldBatch determines if a notification should be batched
func (p *Processor) shouldBatch(notification *models.Notification) bool {
	if notification.Priority == models.PriorityUrgent || notification.Priority == models.PriorityHigh {
		return false
	}
	return p.batchStrategy.ShouldBatch(notification)
}

// deliver delivers a notification to all its channels
func (p *Processor) deliver(ctx context.Context, notification *models.Notification) {
	for _, channel := range notification.Channels {
		go p.deliverToChannel(ctx, notification, channel, 0)
	}
}

// deliverToChannel delivers notification to a specific channel with retry logic
func (p *Processor) deliverToChannel(
	ctx context.Context,
	notification *models.Notification,
	channel models.NotificationChannel,
	retryCount int,
) {
	// Generate deterministic delivery ID using UUID v5
	deliveryIdentifier := fmt.Sprintf("%s:%s:%d",
		notification.ID.String(),
		channel,
		retryCount,
	)
	deliveryID := uuid.NewV5(deliveryNamespace, deliveryIdentifier)

	p.log.Info("Delivering notification",
		"id", notification.ID.String(),
		"channel", channel,
		"retry", retryCount,
		"delivery_id", deliveryID.String(),
	)

	// Create delivery record
	record := &models.DeliveryRecord{
		ID:             deliveryID,
		NotificationID: notification.ID,
		Channel:        channel,
		Status:         models.StatusSending,
		RetryCount:     retryCount,
		LastAttemptAt:  time.Now(),
		CreatedAt:      time.Now(),
	}

	// Attempt delivery
	err := p.simulateDelivery(ctx, channel, notification)

	if err != nil {
		p.log.Error("Delivery failed",
			"error", err,
			"id", notification.ID.String(),
			"channel", channel,
			"retry_count", retryCount,
		)

		record.Status = models.StatusFailed
		record.ErrorMessage = err.Error()

		// Save failed delivery record
		if saveErr := p.repo.SaveDeliveryRecord(ctx, record); saveErr != nil {
			p.log.Error("Failed to save delivery record", "error", saveErr)
		}

		// Retry logic
		if retryCount < p.cfg.MaxRetries {
			delay := p.retryStrategy.CalculateDelay(retryCount)
			p.log.Info("Scheduling retry",
				"id", notification.ID.String(),
				"delay", delay,
				"attempt", retryCount+1,
			)

			p.metrics.mu.Lock()
			p.metrics.TotalRetries++
			p.metrics.mu.Unlock()

			time.AfterFunc(delay, func() {
				p.deliverToChannel(ctx, notification, channel, retryCount+1)
			})
		} else {
			p.log.Error("Max retries exceeded, moving to DLQ", "id", notification.ID.String())
			p.moveToDLQ(notification, channel, err)

			p.metrics.mu.Lock()
			p.metrics.TotalFailed++
			p.metrics.mu.Unlock()
		}
	} else {
		p.log.Info("Notification delivered successfully",
			"id", notification.ID.String(),
			"channel", channel,
		)

		now := time.Now()
		record.Status = models.StatusDelivered
		record.DeliveredAt = &now

		// Save successful delivery record
		if saveErr := p.repo.SaveDeliveryRecord(ctx, record); saveErr != nil {
			p.log.Error("Failed to save delivery record", "error", saveErr)
		}

		p.metrics.mu.Lock()
		p.metrics.TotalDelivered++
		p.metrics.mu.Unlock()
	}
}

// simulateDelivery simulates notification delivery
func (p *Processor) simulateDelivery(
	ctx context.Context,
	channel models.NotificationChannel,
	notification *models.Notification,
) error {
	// If Firebase is configured and channel is PUSH, send real notification
	if channel == models.ChannelPush && p.firebaseService != nil {
		return p.sendRealPushNotification(ctx, notification)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	// Simulate 10% failure rate for demonstration
	if time.Now().UnixNano()%10 == 0 {
		return fmt.Errorf("simulated delivery failure for channel %s", channel)
	}

	return nil
}

// sendRealPushNotification sends actual push notification via Firebase
func (p *Processor) sendRealPushNotification(ctx context.Context, notification *models.Notification) error {
	// In production, fetch device tokens from database
	// For testing, use your test device token
	deviceToken := p.getDeviceToken(notification.UserID)

	if deviceToken == "" {
		p.log.Error("No device token found for user", "user_id", notification.UserID.String())
		return fmt.Errorf("no device token for user")
	}

	return p.firebaseService.SendPushNotification(ctx, notification, deviceToken)
}

// moveToDLQ moves failed notification to Dead Letter Queue
func (p *Processor) moveToDLQ(
	notification *models.Notification,
	channel models.NotificationChannel,
	err error,
) {
	p.log.Error("Moving to DLQ",
		"notification_id", notification.ID.String(),
		"channel", channel,
		"error", err,
	)
	// In production: Store in database table or send to external DLQ service
}

// processBatches periodically processes ready batches
func (p *Processor) processBatches(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			batches := p.batchStrategy.GetReadyBatches()
			if len(batches) > 0 {
				p.log.Info("Processing ready batches", "count", len(batches))

				for _, batch := range batches {
					p.deliverBatch(ctx, batch)
				}
			}
		}
	}
}

// deliverBatch delivers a batch of notifications
func (p *Processor) deliverBatch(ctx context.Context, batch *strategy.Batch) {
	p.log.Info("Delivering batch",
		"user_id", batch.UserID.String(),
		"type", batch.Type,
		"count", len(batch.Notifications),
	)

	// Create a digest notification
	digestNotification := p.createDigestNotification(batch)

	// Deliver the digest
	p.deliver(ctx, digestNotification)
}

// createDigestNotification creates a digest notification from a batch
func (p *Processor) createDigestNotification(batch *strategy.Batch) *models.Notification {
	// Generate digest notification ID
	digestIdentifier := fmt.Sprintf("batch:%s:%s:%d",
		batch.UserID.String(),
		batch.Type,
		time.Now().Unix(),
	)
	digestID := uuid.NewV5(deliveryNamespace, digestIdentifier)

	return &models.Notification{
		ID:             digestID,
		OrganizationID: batch.Notifications[0].OrganizationID,
		UserID:         batch.UserID,
		Type:           batch.Type,
		Priority:       models.PriorityLow,
		Title:          fmt.Sprintf("You have %d new %s notifications", len(batch.Notifications), batch.Type),
		Message:        fmt.Sprintf("Batch digest for %d notifications", len(batch.Notifications)),
		Metadata: map[string]interface{}{
			"batch": true,
			"count": len(batch.Notifications),
		},
		Channels:  []models.NotificationChannel{models.ChannelEmail},
		CreatedAt: time.Now(),
		IsRead:    false,
	}
}

// updateMetrics updates processor metrics
func (p *Processor) updateMetrics(processingTime time.Duration) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	if p.metrics.AverageProcessTime == 0 {
		p.metrics.AverageProcessTime = processingTime
	} else {
		p.metrics.AverageProcessTime = (p.metrics.AverageProcessTime + processingTime) / 2
	}
}

// reportMetrics periodically reports processor metrics
func (p *Processor) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.metrics.mu.RLock()
			p.log.Info("Processor metrics",
				"total_processed", p.metrics.TotalProcessed,
				"total_delivered", p.metrics.TotalDelivered,
				"total_failed", p.metrics.TotalFailed,
				"total_batched", p.metrics.TotalBatched,
				"total_retries", p.metrics.TotalRetries,
				"avg_process_time", p.metrics.AverageProcessTime,
			)
			p.metrics.mu.RUnlock()
		}
	}
}

// GetMetrics returns current processor metrics
func (p *Processor) GetMetrics() map[string]interface{} {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return map[string]interface{}{
		"total_processed":     p.metrics.TotalProcessed,
		"total_delivered":     p.metrics.TotalDelivered,
		"total_failed":        p.metrics.TotalFailed,
		"total_batched":       p.metrics.TotalBatched,
		"total_retries":       p.metrics.TotalRetries,
		"avg_process_time_ms": p.metrics.AverageProcessTime.Milliseconds(),
	}
}

// Shutdown gracefully shuts down the processor
func (p *Processor) Shutdown(ctx context.Context) error {
	p.log.Info("Shutting down processor...")

	// Wait for workers to complete
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("Processor shutdown complete")

		// Log final metrics
		p.metrics.mu.RLock()
		p.log.Info("Final processor metrics",
			"total_processed", p.metrics.TotalProcessed,
			"total_delivered", p.metrics.TotalDelivered,
			"total_failed", p.metrics.TotalFailed,
			"total_batched", p.metrics.TotalBatched,
			"total_retries", p.metrics.TotalRetries,
		)
		p.metrics.mu.RUnlock()

		return nil
	case <-ctx.Done():
		p.log.Error("Processor shutdown timeout")
		return ctx.Err()
	}
}

func (p *Processor) getDeviceToken(userID uuid.UUID) string {
	// In production, fetch device tokens from database
	// For testing, use your test device token
	return "YOUR_TEST_DEVICE_TOKEN"
}
