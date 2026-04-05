package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// ReceivableEntry represents an outstanding receivable tied to an invoice.
type ReceivableEntry struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receivable_entry"`
	// The invoice associated with this receivable entry.
	Invoice *Invoice `json:"invoice"`
	// The customer who owes the receivable.
	Customer *Customer `json:"customer"`
	// The purchase order number on the invoice, if any.
	PONumber *string `json:"po_number"`
	// The date the invoice was created.
	InvoicedAt time.Time `json:"invoiced_at" validate:"required"`
	// The remaining balance on the invoice.
	RemainingBalance string `json:"remaining_balance" validate:"required"`
	// Whether the invoice has been paid in full.
	IsPaidInFull bool `json:"is_paid_in_full"`
}

var SampleReceivableEntry = ReceivableEntry{
	Object: constants.ObjectTypeReceivableEntry,
	Invoice: &Invoice{
		ID:     SampleInvoiceID,
		Object: constants.ObjectTypeInvoice,
		Number: "INV-001",
	},
	Customer: &Customer{
		ID:     SampleCustomerID,
		Object: constants.ObjectTypeCustomer,
		Name:   SampleCustomerName,
		Number: SampleCustomerNumber,
	},
	PONumber:         nil,
	InvoicedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	RemainingBalance: "1234.560000000000000000000000000000",
	IsPaidInFull:     false,
}

func (*ReceivableEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivableEntry)
}
