package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductLineID = "pdln_k9bnlgvxhxjh"
const SampleProductLineName = "Industrial Fasteners"

var sampleProductLineDescription = "Bolts, screws, and anchors for heavy industrial assembly."
var sampleProductLineNotes = "Priced per the 2026 supplier contract; review pricing each quarter."

// A named grouping of related products in your catalog.
//
// A product line carries the default commission and freight policies for the products assigned to it, along with the unit group that determines how those products are measured. Product lines are also the unit that catalog access is granted over, for both customers and account groups.
type ProductLine struct {
	// Product line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_line"`
	// Display name of the product line.
	//
	// Unique among the product lines visible to your account, which includes the shared system lines.
	Name string `json:"name" validate:"required"`
	// Free-form description of the product line.
	Description *string `json:"description"`
	// Free-form notes about the product line.
	Notes *string `json:"notes"`
	// Default commission policy for products in this product line.
	//
	// - `commission_exempt`: no commission applies to these products.
	// - `commission_applied`: commission applies to these products, unless overridden elsewhere.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Default freight policy for products in this product line.
	//
	// - `free_freight`: these products do not incur a freight charge.
	// - `billed_freight`: freight is billed for these products, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Owner of the product line.
	//
	// System-owned product lines are platform-provided and shared across all accounts; account-owned product lines are custom to your account. Only account-owned product lines can be updated, deleted, or granted to customers and account groups.
	Owner *Owner `json:"owner" expandable:"true"`
	// Unit group associated with this product line.
	//
	// The unit group determines the set of units available to products in this product line.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
	// The lot products in this line are made in — a doff, a pallet.
	//
	// Sizes the campaigns a production schedule plans, and defaults the quantity when a batch is added to a production run. The unit is part of the value — 60 counted in pairs and 60 counted in eaches are different lots — and is drawn from this product line's unit group.
	//
	// An item's own lot override still takes precedence over the line's. When the line has no lot convention, planning falls back to the lot of the line the item feeds into, and then to the account-wide default lot size.
	DefaultLot *Quantity `json:"default_lot" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last-updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductLine = &ProductLine{
	ID:               SampleProductLineID,
	Object:           constants.ObjectTypeProductLine,
	Name:             SampleProductLineName,
	Description:      &sampleProductLineDescription,
	Notes:            &sampleProductLineNotes,
	CommissionPolicy: constants.CommissionPolicyExempt,
	FreightPolicy:    constants.FreightPolicyBilled,
	Owner:            SampleOwnerSystem,
	UnitGroup:        SampleUnitGroup,
	DefaultLot:       SampleQuantity,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductLine)
}
