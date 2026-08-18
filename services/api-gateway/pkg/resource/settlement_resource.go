package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSettlementID = "sl_2k5juz0yf5a7"
const SampleSettlementSummaryID = "sl_a21jaxz7ehs1"

// A batch of transaction allocations applying customer payments and credits to invoices.
//
// Each allocation in a settlement applies part of a transaction (payment, rebate, adjustment, or credit memo) to a specific invoice.
type Settlement struct {
	// Settlement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=settlement"`
	// Number identifying the settlement within the account.
	//
	// Generated automatically from a per-account sequence at creation; it can be changed later but must remain unique within the account.
	Number string `json:"number" validate:"required"`
	// Free-form note attached to this settlement.
	Note *string `json:"note"`
	// The account user responsible for this settlement.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// Transaction allocations recorded in this settlement, each applying part of a transaction to one invoice.
	Allocations *List[TransactionAllocation] `json:"allocations" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleSettlementNote = "Applied customer payment across open invoices."

var SampleSettlement = &Settlement{
	ID:              SampleSettlementID,
	Object:          constants.ObjectTypeSettlement,
	Number:          "1",
	Note:            &sampleSettlementNote,
	ResponsibleUser: SampleAccountUser,
	Allocations:     NewList([]TransactionAllocation{*SampleTransactionAllocation2}, PageInfo{}),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Settlement) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSettlement)
}

// A condensed settlement shape returned by List Settlements.
//
// Replaces the full allocation list with aggregate totals per transaction type, plus the invoice numbers and customer names the allocations touch.
//
// When the list is filtered by transaction or invoice, every aggregate here — the allocation count, the totals, the invoice numbers, and the customer names — covers only the allocations that matched the filter, not every allocation in the settlement.
type SettlementSummary struct {
	// Settlement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=settlement_summary"`
	// Settlement number.
	Number string `json:"number" validate:"required"`
	// Number of allocations in this settlement.
	AllocationCount int32 `json:"allocation_count"`
	// Total amount allocated from `payment` transactions, as a decimal string.
	TotalPayments *string `json:"total_payments"`
	// Total amount allocated from `rebate` transactions, as a decimal string.
	TotalRebates *string `json:"total_rebates"`
	// Total amount allocated from `adjustment` transactions, as a decimal string.
	TotalAdjustments *string `json:"total_adjustments"`
	// Total amount allocated from `credit_memo` transactions, as a decimal string.
	TotalCredits *string `json:"total_credits"`
	// Numbers of the invoices this settlement's allocations were applied to, without duplicates.
	InvoiceNumbers []string `json:"invoice_numbers"`
	// Names of the customers billed by those invoices, without duplicates.
	CustomerNames []string `json:"customer_names"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSettlementSummary = &SettlementSummary{
	ID:               SampleSettlementSummaryID,
	Object:           constants.ObjectTypeSettlementSummary,
	Number:           "1",
	AllocationCount:  2,
	TotalPayments:    new("500.000000000000000000000000000000"),
	TotalRebates:     new("0.000000000000000000000000000000"),
	TotalAdjustments: new("0.000000000000000000000000000000"),
	TotalCredits:     new("250.000000000000000000000000000000"),
	InvoiceNumbers:   []string{"INV-001", "INV-002"},
	CustomerNames:    []string{"Acme Corp"},
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SettlementSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSettlementSummary)
}

// A portion of a transaction's amount applied to a specific invoice.
type TransactionAllocation struct {
	// Allocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_allocation"`
	// The part of the transaction's amount applied to the invoice, in US dollars.
	Amount *Quantity `json:"amount" validate:"required"`
	// Free-form note attached to this allocation, separate from any note on the underlying transaction.
	Note *string `json:"note"`
	// Transaction whose amount is being applied to the invoice.
	Transaction *TransactionDetail `json:"transaction" expandable:"true"`
	// The invoice the amount was applied to.
	Invoice *AllocationInvoice `json:"invoice"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleTransactionAllocation2Note = "Applied to the oldest open invoice first."

var SampleTransactionAllocation2 = &TransactionAllocation{
	ID:     SampleAllocationEntryID,
	Object: constants.ObjectTypeTransactionAllocation,
	Amount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "500.000000000000000000000000000000",
		DisplayValue: "$500.00",
		Unit:         newSampleUnit("US Dollar", "$", constants.UnitTypeCurrency),
	},
	Note:        &sampleTransactionAllocation2Note,
	Transaction: SampleTransactionDetail,
	Invoice: &AllocationInvoice{
		ID:     SampleInvoiceID,
		Object: constants.ObjectTypeInvoiceSummary,
		Number: "INV-001",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*TransactionAllocation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionAllocation2)
}
