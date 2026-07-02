package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductLineID = "pl_01996357326a0d3f7b129542ea"
const SampleProductLineName = "Industrial Fasteners"

var sampleProductLineDescription = "Bolts, screws, and anchors for heavy industrial assembly."
var sampleProductLineNotes = "Priced per the 2026 supplier contract; review pricing each quarter."

// Product line resource.
//
// A product line groups related products in your catalog and carries the default commission policy, freight policy, and unit group for those products.
type ProductLine struct {
	// Product line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_line"`
	// Display name of the product line.
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
	// System-owned product lines are platform-provided and shared across all accounts; account-owned product lines are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Unit group associated with this product line.
	//
	// The unit group determines the set of units available to products in this product line.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
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
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductLine)
}
