package strategy

import (
	"fmt"
	"sync"
	"time"

	"github.com/Rizwank123/notification_system/internal/config"
	"github.com/Rizwank123/notification_system/internal/models"
	"github.com/gofrs/uuid/v5"
)

type BatchStrategy struct {
	cfg     *config.Config
	batches map[string]*Batch
	mu      sync.RWMutex
}

type Batch struct {
	UserID        uuid.UUID
	Type          models.NotificationType
	Notifications []*models.Notification
	FirstAdded    time.Time
	LastAdded     time.Time
}

func NewBatchStrategy(cfg *config.Config) *BatchStrategy {
	return &BatchStrategy{
		cfg:     cfg,
		batches: make(map[string]*Batch),
	}
}

func (bs *BatchStrategy) ShouldBatch(notification *models.Notification) bool {
	key := bs.getBatchKey(notification.UserID, string(notification.Type))

	bs.mu.RLock()
	batch, exists := bs.batches[key]
	bs.mu.RUnlock()

	if !exists {
		return false
	}

	windowStart := time.Now().Add(-time.Duration(bs.cfg.BurstWindowSeconds) * time.Second)
	if batch.FirstAdded.After(windowStart) && len(batch.Notifications) >= bs.cfg.BurstThreshold {
		return true
	}

	return false
}

func (bs *BatchStrategy) AddToBatch(notification *models.Notification) {
	key := bs.getBatchKey(notification.UserID, string(notification.Type))

	bs.mu.Lock()
	defer bs.mu.Unlock()

	batch, exists := bs.batches[key]
	if !exists {
		batch = &Batch{
			UserID:        notification.UserID,
			Type:          notification.Type,
			Notifications: make([]*models.Notification, 0),
			FirstAdded:    time.Now(),
		}
		bs.batches[key] = batch
	}

	batch.Notifications = append(batch.Notifications, notification)
	batch.LastAdded = time.Now()
}

func (bs *BatchStrategy) getBatchKey(userID uuid.UUID, notificationType string) string {
	return fmt.Sprintf("%s:%s", userID.String(), notificationType)
}

func (bs *BatchStrategy) GetReadyBatches() []*Batch {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	var ready []*Batch
	batchWindow := time.Duration(bs.cfg.BatchWindowSeconds) * time.Second
	now := time.Now()

	for key, batch := range bs.batches {
		if now.Sub(batch.FirstAdded) >= batchWindow || len(batch.Notifications) >= bs.cfg.BatchSize {
			ready = append(ready, batch)
			delete(bs.batches, key)
		}
	}

	return ready
}
