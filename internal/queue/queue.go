package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Rizwank123/notification_system/internal/models"
	"github.com/Rizwank123/notification_system/pkg/logger"
)

// Queue represents an in-memory notification queue
type Queue struct {
	priorityQueues map[models.NotificationPriority]chan *models.Notification
	mu             sync.RWMutex
	log            logger.Logger
}

func NewQueue(log logger.Logger) *Queue {
	return &Queue{
		priorityQueues: map[models.NotificationPriority]chan *models.Notification{
			models.PriorityUrgent: make(chan *models.Notification, 10000),
			models.PriorityHigh:   make(chan *models.Notification, 10000),
			models.PriorityMedium: make(chan *models.Notification, 10000),
			models.PriorityLow:    make(chan *models.Notification, 10000),
		},
		log: log,
	}
}

func (q *Queue) Enqueue(ctx context.Context, notification *models.Notification) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	queue, exists := q.priorityQueues[notification.Priority]
	if !exists {
		return fmt.Errorf("invalid priority: %s", notification.Priority)
	}

	select {
	case queue <- notification:
		q.log.Debug("Notification enqueued", "id", notification.ID, "priority", notification.Priority)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("queue full for priority: %s", notification.Priority)
	}
}

func (q *Queue) Dequeue(ctx context.Context, priority models.NotificationPriority) (*models.Notification, error) {
	q.mu.RLock()
	queue, exists := q.priorityQueues[priority]
	q.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid priority: %s", priority)
	}

	select {
	case notification := <-queue:
		return notification, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (q *Queue) Size(priority models.NotificationPriority) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if queue, exists := q.priorityQueues[priority]; exists {
		return len(queue)
	}
	return 0
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, queue := range q.priorityQueues {
		close(queue)
	}
}

func MarshalNotification(n *models.Notification) ([]byte, error) {
	return json.Marshal(n)
}

func UnmarshalNotification(data []byte) (*models.Notification, error) {
	var n models.Notification
	err := json.Unmarshal(data, &n)
	return &n, err
}
