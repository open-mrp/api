package constants

// AccountRelationNotificationType defines the types of notifications that can be configured on an account relation.
type AccountRelationNotificationType string

const (
	// AccountRelationNotificationTypeInvoice indicates invoice notifications.
	AccountRelationNotificationTypeInvoice AccountRelationNotificationType = "invoice"
	// AccountRelationNotificationTypeOrderAcknowledgement indicates order acknowledgement notifications.
	AccountRelationNotificationTypeOrderAcknowledgement AccountRelationNotificationType = "order_acknowledgement"
	// AccountRelationNotificationTypeShipment indicates shipment notices.
	AccountRelationNotificationTypeShipment AccountRelationNotificationType = "shipment"
	// AccountRelationNotificationTypeStatement indicates account statements.
	AccountRelationNotificationTypeStatement AccountRelationNotificationType = "statement"
	// AccountRelationNotificationTypePurchaseOrderSubmission indicates purchase order submission notifications.
	AccountRelationNotificationTypePurchaseOrderSubmission AccountRelationNotificationType = "purchase_order_submission"
)

func (m AccountRelationNotificationType) IsValid() bool {
	switch m {
	case AccountRelationNotificationTypeInvoice, AccountRelationNotificationTypeOrderAcknowledgement,
		AccountRelationNotificationTypeShipment, AccountRelationNotificationTypeStatement,
		AccountRelationNotificationTypePurchaseOrderSubmission:
		return true
	default:
		return false
	}
}

func (m AccountRelationNotificationType) EnumValues() []string {
	return []string{
		string(AccountRelationNotificationTypeInvoice),
		string(AccountRelationNotificationTypeOrderAcknowledgement),
		string(AccountRelationNotificationTypeShipment),
		string(AccountRelationNotificationTypeStatement),
		string(AccountRelationNotificationTypePurchaseOrderSubmission),
	}
}

// NotificationType is the type a relation stores for a document its recipients receive.
func (d RecipientDocument) NotificationType() AccountRelationNotificationType {
	if d == RecipientDocumentAcknowledgment {
		return AccountRelationNotificationTypeOrderAcknowledgement
	}
	return AccountRelationNotificationType(d)
}

// RecipientDocument is the document a stored notification type sends, and false for a type that isn't a customer
// document (purchase order submissions).
func (m AccountRelationNotificationType) RecipientDocument() (RecipientDocument, bool) {
	switch m {
	case AccountRelationNotificationTypeOrderAcknowledgement:
		return RecipientDocumentAcknowledgment, true
	case AccountRelationNotificationTypeInvoice, AccountRelationNotificationTypeShipment, AccountRelationNotificationTypeStatement:
		return RecipientDocument(m), true
	default:
		return "", false
	}
}
