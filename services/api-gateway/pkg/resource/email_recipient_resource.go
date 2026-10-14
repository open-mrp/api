package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
)

// One person who receives documents by email, and which ones.
type EmailRecipient struct {
	// The account user who receives the documents. `null` unless included.
	AccountUser *AccountUser `json:"account_user" expandable:"true"`
	// The documents they receive.
	//
	// - `acknowledgment`: the acknowledgment sent when an order is issued.
	// - `invoice`: invoices.
	// - `shipment`: shipment notices, with tracking.
	// - `statement`: account statements.
	Documents []constants.RecipientDocument `json:"documents" validate:"required"`
}

var SampleEmailRecipient = &EmailRecipient{
	Documents: []constants.RecipientDocument{constants.RecipientDocumentAcknowledgment, constants.RecipientDocumentInvoice},
}

func (*EmailRecipient) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailRecipient)
}
