-- +goose Up
-- +goose StatementBegin

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    notification_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL,
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN (
        'TASK_ASSIGNED',
        'TASK_UPDATED',
        'TASK_COMPLETED',
        'TASK_DUE_SOON',
        'PROJECT_CREATED'
    )),
    priority VARCHAR(20) NOT NULL CHECK (priority IN (
        'URGENT',
        'HIGH',
        'MEDIUM',
        'LOW'
    )),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    scheduled_for TIMESTAMP
);

-- Indexes for notifications
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_org_id ON notifications(organization_id);
CREATE INDEX idx_notifications_user_read ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_priority ON notifications(priority);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- Create notification_preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
    preference_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    channel VARCHAR(50) NOT NULL CHECK (channel IN (
        'EMAIL',
        'PUSH',
        'IN_APP',
        'SMS'
    )),
    notification_type VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    delivery_frequency VARCHAR(50) DEFAULT 'INSTANT' CHECK (delivery_frequency IN (
        'INSTANT',
        'BATCHED',
        'HOURLY',
        'DAILY'
    )),
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_user_channel_type UNIQUE (user_id, channel, notification_type)
);

-- Indexes for notification_preferences
CREATE INDEX idx_notif_pref_user_id ON notification_preferences(user_id);
CREATE INDEX idx_notif_pref_channel ON notification_preferences(channel);

-- Create delivery_records table
CREATE TABLE IF NOT EXISTS delivery_records (
    delivery_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL,
    channel VARCHAR(50) NOT NULL CHECK (channel IN (
        'EMAIL',
        'PUSH',
        'IN_APP',
        'SMS'
    )),
    status VARCHAR(50) NOT NULL CHECK (status IN (
        'PENDING',
        'QUEUED',
        'SENDING',
        'DELIVERED',
        'FAILED',
        'BATCHED'
    )),
    retry_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP DEFAULT NOW(),
    delivered_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for delivery_records
CREATE INDEX idx_delivery_notification_id ON delivery_records(notification_id);
CREATE INDEX idx_delivery_status ON delivery_records(status, last_attempt_at);
CREATE INDEX idx_delivery_channel ON delivery_records(channel);
CREATE INDEX idx_delivery_created_at ON delivery_records(created_at DESC);

-- Create dead_letter_queue table
CREATE TABLE IF NOT EXISTS dead_letter_queue (
    dlq_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL,
    channel VARCHAR(50) NOT NULL,
    error_message TEXT NOT NULL,
    retry_count INTEGER NOT NULL,
    last_error_at TIMESTAMP DEFAULT NOW(),
    notification_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP
);

-- Indexes for dead_letter_queue
CREATE INDEX idx_dlq_notification_id ON dead_letter_queue(notification_id);
CREATE INDEX idx_dlq_processed ON dead_letter_queue(processed, created_at);

-- Function to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for notification_preferences
CREATE TRIGGER update_notification_preferences_updated_at
    BEFORE UPDATE ON notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to get unread count
CREATE OR REPLACE FUNCTION get_unread_count(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    unread_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO unread_count
    FROM notifications
    WHERE user_id = p_user_id AND is_read = false;
    RETURN unread_count;
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE notifications IS 'Stores all notifications sent to users';
COMMENT ON TABLE notification_preferences IS 'User notification channel preferences';
COMMENT ON TABLE delivery_records IS 'Tracks notification delivery attempts and status';
COMMENT ON TABLE dead_letter_queue IS 'Failed notifications for manual review';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers
DROP TRIGGER IF EXISTS update_notification_preferences_updated_at ON notification_preferences;

-- Drop functions
DROP FUNCTION IF EXISTS get_unread_count(UUID);
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order
DROP TABLE IF EXISTS dead_letter_queue CASCADE;
DROP TABLE IF EXISTS delivery_records CASCADE;
DROP TABLE IF EXISTS notification_preferences CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;

-- Drop extension (optional, only if not used by other tables)
-- DROP EXTENSION IF EXISTS "uuid-ossp";

-- +goose StatementEnd
