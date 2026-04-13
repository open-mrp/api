package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Outstanding receivable tied to an invoice.
type ReceivableEntry struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receivable_entry"`
	// Associated invoice.
	Invoice *Invoice `json:"invoice"`
	// Customer who owes the receivable.
	Customer *Customer `json:"customer"`
	// Purchase order number, if any.
	PONumber *string `json:"po_number"`
	// Invoice creation date.
	InvoicedAt time.Time `json:"invoiced_at" validate:"required"`
	// Remaining balance on the invoice.
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
