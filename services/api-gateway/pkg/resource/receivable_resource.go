package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// An outstanding balance owed on an invoice.
//
// Receivable entries are derived from invoices that have not been paid in full; one entry is returned per open invoice.
type ReceivableEntry struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=receivable_entry"`
	// The invoice the balance is owed on.
	//
	// Only the invoice's identifying fields (`id`, `number`) are populated.
	Invoice *Invoice `json:"invoice"`
	// Customer who owes the balance.
	//
	// Only the customer's identifying fields (`id`, `name`, `number`) are populated.
	Customer *Customer `json:"customer"`
	// Customer's purchase order number from the underlying sales order, if any.
	PONumber *string `json:"po_number"`
	// Invoice creation date.
	InvoicedAt time.Time `json:"invoiced_at" validate:"required"`
	// Remaining unpaid balance on the invoice, as a decimal string.
	//
	// Calculated as the invoiced total minus all transaction allocations applied to the invoice. When a `cutoff_date` is supplied to the listing endpoint, only allocations made before that date are subtracted.
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
