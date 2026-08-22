package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// A default order-notification recipient for a customer.
//
// Each recipient pairs an account user on the customer's own account with the notification types they receive for that customer's orders.
type OrderNotificationRecipient struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=order_notification_recipient"`
	// The account user who receives these notifications.
	//
	// The account user's `user` profile is not returned here; resolve the recipient's name and email from the account users on the customer's own account.
	AccountUser *AccountUser `json:"account_user" expandable:"true"`
	// Order notification types this recipient receives.
	//
	// - `order_acknowledgement`: the confirmation email sent when an order is placed for the customer.
	// - `invoice`: invoice emails for the customer's orders.
	// - `purchase_order_submission`: a copy of each purchase order you submit to this account as a supplier; those recipients are managed alongside the account's users and cannot be set through the customer notification-recipient endpoints.
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
