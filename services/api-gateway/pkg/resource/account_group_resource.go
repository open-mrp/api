package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleAccountGroupID = "acgp_6p4z57e9alaf"
const SampleAccountGroupName = "Wholesale Customers"

// A group that classifies customers, such as `Distributors`, and sets policies and defaults for every customer in it.
type AccountGroup struct {
	// Account group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group"`
	// How the group is used: `type_group` classifies customers, `pricing_group` scopes pricing.
	Type constants.AccountGroupType `json:"type" validate:"required" visibility:"internal"`
	// Display name, unique (case-insensitive) among your account's groups.
	Name string `json:"name" validate:"required"`
	// Free-form description of the group.
	Description *string `json:"description"`
	// Whether orders from the group's customers earn commission.
	//
	// A customer is exempt if it or its group is `exempt`. Always `null` for portal users.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy" validate:"required" sensitive:"internal"`
	// Whether the group's customers are billed for freight.
	//
	// Freight is waived if the customer, its group, or the product line of an ordered item is `free`.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Values the group's customers inherit when they don't set their own.
	Defaults *AccountGroupDefaults `json:"defaults" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Values an account group's customers inherit when they don't set their own.
type AccountGroupDefaults struct {
	// Lead time for customers in this group that don't set their own or inherit one from a parent customer.
	//
	// Business days from an order being issued to it being due to ship. `null` uses your account default.
	LeadTime *Quantity `json:"lead_time"`
}

var SampleAccountGroup = &AccountGroup{
	ID:               SampleAccountGroupID,
	Object:           constants.ObjectTypeAccountGroup,
	Type:             constants.AccountGroupTypeTypeGroup,
	Name:             SampleAccountGroupName,
	Description:      nil,
	CommissionPolicy: new(constants.CommissionPolicyApplied),
	FreightPolicy:    constants.FreightPolicyBilled,
	Defaults:         &AccountGroupDefaults{LeadTime: &Quantity{Value: "5", Unit: SampleDayUnit}},
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountGroup)
}
