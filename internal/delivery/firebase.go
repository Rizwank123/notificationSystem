package delivery

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/Rizwank123/notification_system/internal/config"
	"github.com/Rizwank123/notification_system/internal/models"
	"github.com/Rizwank123/notification_system/pkg/logger"
	"google.golang.org/api/option"
)

type FirebaseService struct {
	client *messaging.Client
	log    logger.Logger
}

func NewFirebaseService(ctx context.Context, cfg *config.Config, log logger.Logger) (*FirebaseService, error) {
	if cfg.FirebaseProjectID == "" || cfg.FirebaseCredentialPath == "" {
		return nil, fmt.Errorf("firebase credentials not configured")
	}

	// ✅ FIX: Pass Firebase config with ProjectID
	firebaseConfig := &firebase.Config{
		ProjectID: cfg.FirebaseProjectID, // ← This was missing!
	}

	opt := option.WithCredentialsFile(cfg.FirebaseCredentialPath)

	// Pass firebaseConfig as first parameter
	app, err := firebase.NewApp(ctx, firebaseConfig, opt) // ← Changed from nil to firebaseConfig
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %w", err)
	}

	log.Info("✅ Firebase service initialized successfully", "project_id", cfg.FirebaseProjectID)

	return &FirebaseService{
		client: client,
		log:    log,
	}, nil
}

// SendPushNotification sends a push notification via FCM
func (f *FirebaseService) SendPushNotification(ctx context.Context, notification *models.Notification, deviceToken string) error {
	f.log.Info("📱 Sending FCM notification",
		"notification_id", notification.ID.String(),
		"device_token", deviceToken[:20]+"...",
	)

	// Build FCM message
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Message,
		},
		Data: map[string]string{
			"notification_id": notification.ID.String(),
			"type":            string(notification.Type),
			"priority":        string(notification.Priority),
		},
		Token: deviceToken,
	}

	// Add Android-specific config
	message.Android = &messaging.AndroidConfig{
		Priority: f.getAndroidPriority(notification.Priority),
		Notification: &messaging.AndroidNotification{
			Title:    notification.Title,
			Body:     notification.Message,
			Sound:    "default",
			Priority: messaging.PriorityHigh,
		},
	}

	// Add iOS-specific config
	message.APNS = &messaging.APNSConfig{
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Alert: &messaging.ApsAlert{
					Title: notification.Title,
					Body:  notification.Message,
				},
				Badge: f.getBadgeCount(),
				Sound: "default",
			},
		},
	}

	// Send message
	response, err := f.client.Send(ctx, message)
	if err != nil {
		f.log.Error("❌ Failed to send FCM notification",
			"error", err,
			"notification_id", notification.ID.String(),
		)
		return fmt.Errorf("fcm send failed: %w", err)
	}

	f.log.Info("✅ FCM notification sent successfully",
		"notification_id", notification.ID.String(),
		"message_id", response,
	)

	return nil
}

// SendMulticast sends to multiple devices
func (f *FirebaseService) SendMulticast(ctx context.Context, notification *models.Notification, deviceTokens []string) error {
	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: notification.Title,
			Body:  notification.Message,
		},
		Data: map[string]string{
			"notification_id": notification.ID.String(),
			"type":            string(notification.Type),
		},
		Tokens: deviceTokens,
	}

	response, err := f.client.SendMulticast(ctx, message)
	if err != nil {
		return fmt.Errorf("fcm multicast failed: %w", err)
	}

	f.log.Info("FCM multicast sent",
		"success_count", response.SuccessCount,
		"failure_count", response.FailureCount,
	)

	// Log failures
	if response.FailureCount > 0 {
		for idx, resp := range response.Responses {
			if !resp.Success {
				f.log.Error("FCM send failed for device",
					"token", deviceTokens[idx],
					"error", resp.Error,
				)
			}
		}
	}

	return nil
}

func (f *FirebaseService) getAndroidPriority(priority models.NotificationPriority) string {
	switch priority {
	case models.PriorityUrgent, models.PriorityHigh:
		return "high"
	default:
		return "normal"
	}
}

func (f *FirebaseService) getBadgeCount() *int {
	// In production, fetch from database
	count := 1
	return &count
}
