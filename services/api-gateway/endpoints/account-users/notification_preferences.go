package accountuserep

import "github.com/augno/api/shared/constants"

// NotificationPreferenceItem toggles a single account-relation notification type.
type NotificationPreferenceItem struct {
	// Notification type code. Must match a value of constants.AccountRelationNotificationType.
	NotificationTypeCode constants.AccountRelationNotificationType `json:"notification_type_code" validate:"required,oneof=invoice order_acknowledgement purchase_order_submission"`
	// Whether this notification type is enabled for the account user.
	Enabled bool `json:"enabled"`
}
