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
	// Purchase order number the customer supplied on the underlying sales order.
	PONumber *string `json:"po_number"`
	// Date the invoice was created.
	InvoicedAt time.Time `json:"invoiced_at" validate:"required"`
	// Remaining unpaid balance on the invoice.
	//
	// Calculated as the invoiced total minus all transaction allocations applied to the invoice. When a `cutoff_at` is supplied to the listing endpoint, only allocations made before that date are subtracted.
	RemainingBalance string `json:"remaining_balance" validate:"required"`
	// Whether the invoice has been paid in full.
	//
	// Always `false` here, because only invoices that still owe a balance produce a receivable entry.
	IsPaidInFull bool `json:"is_paid_in_full"`
}

var SampleReceivableEntry = ReceivableEntry{
	Object:           constants.ObjectTypeReceivableEntry,
	Invoice:          SampleInvoice,
	Customer:         SampleCustomer,
	PONumber:         nil,
	InvoicedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	RemainingBalance: "1234.56",
	IsPaidInFull:     false,
}

func (*ReceivableEntry) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReceivableEntry)
}
