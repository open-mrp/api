package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleProductLineID = "pdln_k9bnlgvxhxjh"
const SampleProductLineName = "Industrial Fasteners"

var sampleProductLineDescription = "Bolts, screws, and anchors for heavy industrial assembly."

// The commercial grouping of items: which customers may buy them, and the commission, freight and supply defaults they're sold under.
type ProductLine struct {
	// Product line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_line"`
	// Display name, such as `Outdoor`.
	//
	// Unique (case-insensitive) among your account's lines and the system lines.
	Name string `json:"name" validate:"required"`
	// Description of the line, shown to customers in the portal.
	Description *string `json:"description"`
	// Whether the line can be newly assigned.
	//
	// `active` or `archived`. Archived lines stay on the products already in them, but can't be assigned to new products or granted to customers.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Whether sales of this line's products earn commission.
	//
	// `applied` or `exempt`. A customer's or account group's policy can override it. Always `null` for portal users.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy" validate:"required" sensitive:"internal"`
	// Whether this line's products add a freight charge.
	//
	// `billed` or `free`. A customer's or account group's policy can override it.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Where this line's items come from when they don't say for themselves.
	//
	// `stock`: taken from inventory. `make`: produced for the order. `buy`: purchased for the order. `null` uses the account default. Always `null` for portal users.
	Supply *constants.Supply `json:"supply" sensitive:"internal"`
	// Whether the line is a system line every account shares or one your account created.
	//
	// System lines can't be updated, deleted, or granted to customers and account groups.
	Owner *Owner `json:"owner" expandable:"true"`
	// Unit group the line's products are priced in, until items carry their own order units.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true" visibility:"internal"`
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
	Status:           constants.ArchiveStatusActive,
	CommissionPolicy: new(constants.CommissionPolicyExempt),
	FreightPolicy:    constants.FreightPolicyBilled,
	Supply:           new(constants.SupplyStock),
	Owner:            SampleOwnerSystem,
	UnitGroup:        SampleUnitGroup,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductLine)
}
