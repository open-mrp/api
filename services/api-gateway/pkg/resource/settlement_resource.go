package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSettlementID = "sl_014f3f9af18ff1c8ded3205149"
const SampleSettlementSummaryID = "sl_01b853556dc1a635122ebbb761"

// Settlement with expandable allocations.
type Settlement struct {
	// Settlement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=settlement"`
	// Settlement number.
	Number string `json:"number" validate:"required"`
	// Note attached to this settlement.
	Note *string `json:"note"`
	// Responsible user.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// Transaction allocations in this settlement.
	Allocations *List[TransactionAllocation] `json:"allocations" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSettlement = &Settlement{
	ID:     SampleSettlementID,
	Object: constants.ObjectTypeSettlement,
	Number: "1",
	ResponsibleUser: &AccountUser{
		ID:     SampleAccountUserID,
		Object: constants.ObjectTypeAccountUser,
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Settlement) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSettlement)
}

// SettlementSummary is a lightweight settlement for list views.
type SettlementSummary struct {
	// Settlement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=settlement_summary"`
	// Settlement number.
	Number string `json:"number" validate:"required"`
	// Number of allocations in this settlement.
	AllocationCount int32 `json:"allocation_count"`
	// Total payment amount as a decimal string.
	TotalPayments *string `json:"total_payments"`
	// Total rebate amount as a decimal string.
	TotalRebates *string `json:"total_rebates"`
	// Total adjustment amount as a decimal string.
	TotalAdjustments *string `json:"total_adjustments"`
	// Total credit amount as a decimal string.
	TotalCredits *string `json:"total_credits"`
	// Invoice numbers included in this settlement.
	InvoiceNumbers []string `json:"invoice_numbers"`
	// Customer names included in this settlement.
	CustomerNames []string `json:"customer_names"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSettlementSummary = &SettlementSummary{
	ID:              SampleSettlementSummaryID,
	Object:          constants.ObjectTypeSettlementSummary,
	Number:          "1",
	AllocationCount: 2,
	InvoiceNumbers:  []string{"INV-001", "INV-002"},
	CustomerNames:   []string{"Acme Corp"},
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SettlementSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSettlementSummary)
}

// Allocation of a transaction against an invoice.
type TransactionAllocation struct {
	// Allocation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_allocation"`
	// Allocated amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// Note.
	Note *string `json:"note"`
	// Associated transaction. Expandable via include[]=allocations.transaction.
	Transaction *TransactionDetail `json:"transaction" expandable:"true"`
	// Associated invoice.
	Invoice *InvoiceSummary `json:"invoice"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleTransactionAllocation2 = &TransactionAllocation{
	ID:     SampleAllocationEntryID,
	Object: constants.ObjectTypeTransactionAllocation,
	Amount: &Quantity{
		ID:           SampleQuantityID,
		Object:       constants.ObjectTypeQuantity,
		Value:        "500.000000000000000000000000000000",
		DisplayValue: "$500.00",
		Unit: &Unit{
			ID:           SampleUnitID,
			Object:       constants.ObjectTypeUnit,
			Name:         "US Dollar",
			Abbreviation: "$",
			Type:         constants.UnitTypeCurrency,
		},
	},
	Transaction: SampleTransactionDetail,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*TransactionAllocation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionAllocation2)
}
