package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Rizwank123/notification_system/internal/config"
	"github.com/Rizwank123/notification_system/internal/delivery"
	"github.com/Rizwank123/notification_system/internal/models"
	"github.com/Rizwank123/notification_system/internal/processor"
	"github.com/Rizwank123/notification_system/internal/queue"
	"github.com/Rizwank123/notification_system/internal/repository"
	"github.com/Rizwank123/notification_system/pkg/logger"
	"github.com/gofrs/uuid/v5"
)

type NotificationService struct {
	cfg       *config.Config
	queue     *queue.Queue
	processor *processor.Processor
	repo      *repository.NotificationRepository
	log       logger.Logger
}

func NewNotificationService(
	cfg *config.Config,
	log logger.Logger,
	repo *repository.NotificationRepository,
	firebaseService *delivery.FirebaseService,
) *NotificationService {
	q := queue.NewQueue(log)
	proc := processor.NewProcessor(cfg, q, log, repo, firebaseService)

	return &NotificationService{
		cfg:       cfg,
		queue:     q,
		processor: proc,
		repo:      repo,
		log:       log,
	}
}

func (ns *NotificationService) Start(ctx context.Context) error {
	ns.log.Info("Starting notification service")

	if err := ns.processor.Start(ctx); err != nil {
		return err
	}

	go ns.simulateBurstScenario(ctx)

	return nil
}

func (ns *NotificationService) SendNotification(ctx context.Context, notification *models.Notification) error {
	// Save to database
	if err := ns.repo.SaveNotification(ctx, notification); err != nil {
		ns.log.Error("Failed to save notification to database", "error", err)
		// Continue even if database save fails
	}

	// Enqueue for processing
	return ns.queue.Enqueue(ctx, notification)
}

func (ns *NotificationService) simulateBurstScenario(ctx context.Context) {
	ns.log.Info("Starting burst scenario simulation: 40,000 notifications in 1 minute")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	notificationCount := 0
	targetCount := 2
	batchSize := 2

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if notificationCount >= targetCount {
				ns.log.Info("Burst scenario completed", "total_sent", notificationCount)
				return
			}

			for i := 0; i < batchSize && notificationCount < targetCount; i++ {
				notification := ns.generateSampleNotification(notificationCount)
				if err := ns.SendNotification(ctx, notification); err != nil {
					ns.log.Error("Failed to send notification", "error", err)
				}
				notificationCount++
			}

			ns.log.Info("Burst progress", "sent", notificationCount, "target", targetCount)
		}
	}
}

func (ns *NotificationService) generateSampleNotification(index int) *models.Notification {
	priorities := []models.NotificationPriority{
		models.PriorityUrgent,
		models.PriorityHigh,
		models.PriorityMedium,
		models.PriorityLow,
	}

	types := []models.NotificationType{
		models.TypeTaskAssigned,
		models.TypeTaskUpdated,
		models.TypeTaskCompleted,
		models.TypeTaskDueSoon,
	}

	return &models.Notification{
		ID:             uuid.Must(uuid.NewV4()),
		OrganizationID: uuid.Must(uuid.NewV4()),
		UserID:         uuid.Must(uuid.NewV4()),
		Type:           types[index%len(types)],
		Priority:       priorities[index%len(priorities)],
		Title:          "Task Notification",
		Message:        "You have a new task update",
		Metadata: map[string]interface{}{
			"task_id": fmt.Sprintf("task-%d", index),
		},
		Channels:  []models.NotificationChannel{models.ChannelPush},
		CreatedAt: time.Now(),
	}
}

func (ns *NotificationService) Shutdown(ctx context.Context) error {
	ns.log.Info("Shutting down notification service")

	if err := ns.processor.Shutdown(ctx); err != nil {
		ns.log.Error("Error shutting down processor", "error", err)
	}

	ns.queue.Close()

	ns.log.Info("Notification service shutdown complete")
	return nil
}
