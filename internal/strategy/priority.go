package strategy

import (
	"github.com/Rizwank123/notification_system/internal/models"
)

type PriorityStrategy struct{}

func NewPriorityStrategy() *PriorityStrategy {
	return &PriorityStrategy{}
}

func (ps *PriorityStrategy) DeterminePriority(notificationType models.NotificationType, metadata map[string]interface{}) models.NotificationPriority {
	if ps.isUrgent(notificationType, metadata) {
		return models.PriorityUrgent
	}

	switch notificationType {
	case models.TypeTaskDueSoon:
		return models.PriorityUrgent
	case models.TypeTaskAssigned:
		return models.PriorityHigh
	case models.TypeTaskUpdated:
		return models.PriorityMedium
	case models.TypeTaskCompleted:
		return models.PriorityLow
	case models.TypeProjectCreated:
		return models.PriorityMedium
	default:
		return models.PriorityLow
	}
}

func (ps *PriorityStrategy) isUrgent(notificationType models.NotificationType, metadata map[string]interface{}) bool {
	if dueIn, ok := metadata["due_in_hours"]; ok {
		if hours, ok := dueIn.(float64); ok && hours <= 1 {
			return true
		}
	}

	if urgent, ok := metadata["is_urgent"]; ok {
		if isUrgent, ok := urgent.(bool); ok && isUrgent {
			return true
		}
	}

	return false
}
