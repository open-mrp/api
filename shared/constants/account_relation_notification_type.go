package constants

// AccountRelationNotificationType defines the types of notifications that can
// be configured on an account relation.
type AccountRelationNotificationType string

const (
	// AccountRelationNotificationTypeInvoice indicates invoice notifications.
	AccountRelationNotificationTypeInvoice AccountRelationNotificationType = "invoice"
	// AccountRelationNotificationTypeOrderAcknowledgement indicates order acknowledgement notifications.
	AccountRelationNotificationTypeOrderAcknowledgement AccountRelationNotificationType = "order_acknowledgement"
	// AccountRelationNotificationTypePurchaseOrderSubmission indicates purchase order submission notifications.
	AccountRelationNotificationTypePurchaseOrderSubmission AccountRelationNotificationType = "purchase_order_submission"
)

func (m AccountRelationNotificationType) IsValid() bool {
	switch m {
	case AccountRelationNotificationTypeInvoice, AccountRelationNotificationTypeOrderAcknowledgement, AccountRelationNotificationTypePurchaseOrderSubmission:
		return true
	default:
		return false
	}
}

func (m AccountRelationNotificationType) EnumValues() []string {
	return []string{
		string(AccountRelationNotificationTypeInvoice),
		string(AccountRelationNotificationTypeOrderAcknowledgement),
		string(AccountRelationNotificationTypePurchaseOrderSubmission),
	}
}
