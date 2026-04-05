package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSettlementID = "sl_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSettlementSummaryID = "sl_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleTransactionAllocationID2 = "txal_01jm4r67aab8nwq3v5hx2d9ktp"

// Settlement represents a full settlement with expandable allocations.
type Settlement struct {
	// The unique identifier for the settlement.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=settlement"`
	// The settlement number.
	Number string `json:"number" validate:"required"`
	// A note attached to this settlement.
	Note *string `json:"note"`
	// The user responsible for this settlement.
	ResponsibleUser *AccountUser `json:"responsible_user" expandable:"true"`
	// The transaction allocations in this settlement.
	Allocations *List[TransactionAllocation] `json:"allocations" expandable:"true"`
	// The timestamp when the settlement was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the settlement was last updated.
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

// SettlementSummary represents a lightweight settlement for list views.
type SettlementSummary struct {
	// The unique identifier for the settlement.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=settlement_summary"`
	// The settlement number.
	Number string `json:"number" validate:"required"`
	// The number of allocations in this settlement.
	AllocationCount int32 `json:"allocation_count"`
	// The total payment amount as a decimal string.
	TotalPayments *string `json:"total_payments"`
	// The total rebate amount as a decimal string.
	TotalRebates *string `json:"total_rebates"`
	// The total adjustment amount as a decimal string.
	TotalAdjustments *string `json:"total_adjustments"`
	// The total credit amount as a decimal string.
	TotalCredits *string `json:"total_credits"`
	// The invoice numbers included in this settlement.
	InvoiceNumbers []string `json:"invoice_numbers"`
	// The customer names included in this settlement.
	CustomerNames []string `json:"customer_names"`
	// The timestamp when the settlement was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the settlement was last updated.
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

// TransactionAllocation represents an allocation of a transaction against an invoice.
type TransactionAllocation struct {
	// The unique identifier for the allocation.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=transaction_allocation"`
	// The allocated amount.
	Amount *Quantity `json:"amount" validate:"required"`
	// A note about this allocation.
	Note *string `json:"note"`
	// The transaction associated with this allocation.
	Transaction *Transaction `json:"transaction" validate:"required"`
	// The invoice associated with this allocation.
	Invoice *InvoiceSummary `json:"invoice"`
	// The timestamp when the allocation was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the allocation was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleTransactionAllocation2 = &TransactionAllocation{
	ID:     SampleTransactionAllocationID2,
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
	Transaction: SampleTransaction,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*TransactionAllocation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTransactionAllocation2)
}
