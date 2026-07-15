package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// A default order-notification recipient for a customer: an account user on the customer's account and the order notification types they receive on that customer's orders.
type OrderNotificationRecipient struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_notification_recipient"`
	// The account user who receives these notifications.
	AccountUser *AccountUser `json:"account_user" expandable:"true"`
	// Order notification types this recipient receives.
	NotificationTypes []constants.AccountRelationNotificationType `json:"notification_types" validate:"required"`
}

var SampleOrderNotificationRecipient = &OrderNotificationRecipient{
	Object:      constants.ObjectTypeOrderNotificationRecipient,
	AccountUser: SampleAccountUser,
	NotificationTypes: []constants.AccountRelationNotificationType{
		constants.AccountRelationNotificationTypeOrderAcknowledgement,
		constants.AccountRelationNotificationTypeInvoice,
	},
}

func (*OrderNotificationRecipient) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOrderNotificationRecipient)
}
