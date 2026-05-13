package accountuserep

import "github.com/augno/api/shared/constants"

// NotificationPreferenceItem toggles a single account-relation notification type.
type NotificationPreferenceItem struct {
	// Notification type.
	NotificationTypeCode constants.AccountRelationNotificationType `json:"notification_type" validate:"required"`
	// Whether this notification type is enabled for the account user.
	Enabled bool `json:"enabled"`
}
