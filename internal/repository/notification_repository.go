package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Rizwank123/notification_system/internal/models"
	"github.com/Rizwank123/notification_system/pkg/logger"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
	log  logger.Logger
}

func NewNotificationRepository(pool *pgxpool.Pool, log logger.Logger) *NotificationRepository {
	return &NotificationRepository{
		pool: pool,
		log:  log,
	}
}

// SaveNotification saves a notification to the database
func (r *NotificationRepository) SaveNotification(ctx context.Context, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (
			notification_id, organization_id, user_id, type, priority, 
			title, message, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	metadata, err := json.Marshal(notification.Metadata)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(
		ctx, query,
		notification.ID,
		notification.OrganizationID,
		notification.UserID,
		notification.Type,
		notification.Priority,
		notification.Title,
		notification.Message,
		metadata,
		notification.CreatedAt,
	)

	if err != nil {
		r.log.Error("Failed to save notification", "error", err, "id", notification.ID)
		return err
	}

	r.log.Debug("Notification saved to database", "id", notification.ID)
	return nil
}

// GetNotificationsByUser retrieves notifications for a specific user
func (r *NotificationRepository) GetNotificationsByUser(ctx context.Context, userID string, limit int) ([]*models.Notification, error) {
	query := `
		SELECT notification_id, organization_id, user_id, type, priority, 
		       title, message, metadata, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		var n models.Notification
		var metadataJSON []byte

		err := rows.Scan(
			&n.ID,
			&n.OrganizationID,
			&n.UserID,
			&n.Type,
			&n.Priority,
			&n.Title,
			&n.Message,
			&metadataJSON,
			&n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
				return nil, err
			}
		}

		notifications = append(notifications, &n)
	}

	return notifications, rows.Err()
}

// SaveDeliveryRecord saves delivery record
func (r *NotificationRepository) SaveDeliveryRecord(ctx context.Context, record *models.DeliveryRecord) error {
	query := `
		INSERT INTO delivery_records (
			delivery_id, notification_id, channel, status, retry_count,
			last_attempt_at, delivered_at, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (delivery_id) DO UPDATE SET
			status = EXCLUDED.status,
			retry_count = EXCLUDED.retry_count,
			last_attempt_at = EXCLUDED.last_attempt_at,
			delivered_at = EXCLUDED.delivered_at,
			error_message = EXCLUDED.error_message
	`

	_, err := r.pool.Exec(
		ctx, query,
		record.ID,
		record.NotificationID,
		record.Channel,
		record.Status,
		record.RetryCount,
		record.LastAttemptAt,
		record.DeliveredAt,
		record.ErrorMessage,
	)

	return err
}

// GetUserPreferences retrieves user notification preferences
func (r *NotificationRepository) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*models.UserPreference, error) {
	query := `
		SELECT user_id, channel, notification_type, is_enabled, delivery_frequency,
		       quiet_hours_start, quiet_hours_end
		FROM notification_preferences
		WHERE user_id = $1
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pref := &models.UserPreference{
		UserID:             userID,
		ChannelPreferences: make(map[models.NotificationChannel]models.ChannelPref),
	}

	for rows.Next() {
		var (
			uid             string
			channel         string
			notifType       string
			enabled         bool
			frequency       string
			quietHoursStart *time.Time
			quietHoursEnd   *time.Time
		)

		err := rows.Scan(
			&uid,
			&channel,
			&notifType,
			&enabled,
			&frequency,
			&quietHoursStart,
			&quietHoursEnd,
		)
		if err != nil {
			return nil, err
		}

		pref.ChannelPreferences[models.NotificationChannel(channel)] = models.ChannelPref{
			Enabled:           enabled,
			DeliveryFrequency: frequency,
		}

		if quietHoursStart != nil {
			pref.QuietHoursStart = quietHoursStart
		}
		if quietHoursEnd != nil {
			pref.QuietHoursEnd = quietHoursEnd
		}
	}

	return pref, rows.Err()
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, notificationID string) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE notification_id = $2
	`

	_, err := r.pool.Exec(ctx, query, time.Now(), notificationID)
	return err
}

// GetUnreadCount returns unread notification count for a user
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1 AND is_read = false
	`

	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// BatchInsertNotifications inserts multiple notifications in a single transaction
func (r *NotificationRepository) BatchInsertNotifications(ctx context.Context, notifications []*models.Notification) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	query := `
		INSERT INTO notifications (
			notification_id, organization_id, user_id, type, priority, 
			title, message, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, notification := range notifications {
		metadata, err := json.Marshal(notification.Metadata)
		if err != nil {
			return err
		}

		batch.Queue(
			query,
			notification.ID,
			notification.OrganizationID,
			notification.UserID,
			notification.Type,
			notification.Priority,
			notification.Title,
			notification.Message,
			metadata,
			notification.CreatedAt,
		)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	for range notifications {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
