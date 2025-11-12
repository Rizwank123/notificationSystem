package models

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// Notification Types
type NotificationType string

const (
	TypeTaskAssigned   NotificationType = "TASK_ASSIGNED"
	TypeTaskUpdated    NotificationType = "TASK_UPDATED"
	TypeTaskCompleted  NotificationType = "TASK_COMPLETED"
	TypeTaskDueSoon    NotificationType = "TASK_DUE_SOON"
	TypeProjectCreated NotificationType = "PROJECT_CREATED"
)

// Notification Priorities
type NotificationPriority string

const (
	PriorityUrgent NotificationPriority = "URGENT"
	PriorityHigh   NotificationPriority = "HIGH"
	PriorityMedium NotificationPriority = "MEDIUM"
	PriorityLow    NotificationPriority = "LOW"
)

// Delivery Channels
type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "EMAIL"
	ChannelPush  NotificationChannel = "PUSH"
	ChannelInApp NotificationChannel = "IN_APP"
	ChannelSMS   NotificationChannel = "SMS"
)

// Delivery Status
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "PENDING"
	StatusQueued    DeliveryStatus = "QUEUED"
	StatusSending   DeliveryStatus = "SENDING"
	StatusDelivered DeliveryStatus = "DELIVERED"
	StatusFailed    DeliveryStatus = "FAILED"
	StatusBatched   DeliveryStatus = "BATCHED"
)

// Notification represents a notification to be sent
type Notification struct {
	ID             uuid.UUID              `json:"id" db:"notification_id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	Type           NotificationType       `json:"type" db:"type"`
	Priority       NotificationPriority   `json:"priority" db:"priority"`
	Title          string                 `json:"title" db:"title"`
	Message        string                 `json:"message" db:"message"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Channels       []NotificationChannel  `json:"channels" db:"-"`
	IsRead         bool                   `json:"is_read" db:"is_read"`
	ReadAt         *time.Time             `json:"read_at,omitempty" db:"read_at"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	ScheduledFor   *time.Time             `json:"scheduled_for,omitempty" db:"scheduled_for"`
}

// UserPreference represents user notification preferences
type UserPreference struct {
	UserID             uuid.UUID                           `json:"user_id" db:"user_id"`
	ChannelPreferences map[NotificationChannel]ChannelPref `json:"channel_preferences"`
	QuietHoursStart    *time.Time                          `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
	QuietHoursEnd      *time.Time                          `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`
	BatchingEnabled    bool                                `json:"batching_enabled" db:"batching_enabled"`
}

// ChannelPref represents preferences for a notification channel
type ChannelPref struct {
	Enabled           bool   `json:"enabled" db:"is_enabled"`
	DeliveryFrequency string `json:"delivery_frequency" db:"delivery_frequency"` // INSTANT, BATCHED, HOURLY, DAILY
}

// DeliveryRecord represents a notification delivery attempt
type DeliveryRecord struct {
	ID             uuid.UUID           `json:"id" db:"delivery_id"`
	NotificationID uuid.UUID           `json:"notification_id" db:"notification_id"`
	Channel        NotificationChannel `json:"channel" db:"channel"`
	Status         DeliveryStatus      `json:"status" db:"status"`
	RetryCount     int                 `json:"retry_count" db:"retry_count"`
	LastAttemptAt  time.Time           `json:"last_attempt_at" db:"last_attempt_at"`
	DeliveredAt    *time.Time          `json:"delivered_at,omitempty" db:"delivered_at"`
	ErrorMessage   string              `json:"error_message,omitempty" db:"error_message"`
	CreatedAt      time.Time           `json:"created_at" db:"created_at"`
}

// TaskEvent represents an event that triggers notifications
type TaskEvent struct {
	EventType      string                 `json:"event_type"`
	TaskID         uuid.UUID              `json:"task_id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	ActorID        uuid.UUID              `json:"actor_id"`
	Recipients     []uuid.UUID            `json:"recipients"`
	Priority       string                 `json:"priority"`
	Metadata       map[string]interface{} `json:"metadata"`
	Timestamp      time.Time              `json:"timestamp"`
}
