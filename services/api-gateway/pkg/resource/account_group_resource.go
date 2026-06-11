package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountGroupID = "acgp_018e88072d1320808dc979cfac"
const SampleAccountGroupName = "Wholesale Customers"

// Account group resource.
type AccountGroup struct {
	// Account group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Free-form description of the account group.
	//
	// Optional; `null` when not set.
	Description *string `json:"description"`
	// Commission policy.
	//
	// - `commission_exempt`: no commission applies.
	// - `commission_applied`: commission applies; if the account group is within a sales rep's territory, it will be assigned to that rep unless overridden.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Freight policy.
	//
	// - `free_freight`: customers within this group will not have to pay for freight.
	// - `billed_freight`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Account group type.
	//
	// - `pricing_group`: used for pricing rules, such as a "Preferred" group that receives a special discount.
	// - `type_group`: used to categorize accounts, such as "Consumers" or "Distributors".
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountGroup = &AccountGroup{
	ID:               SampleAccountGroupID,
	Object:           constants.ObjectTypeAccountGroup,
	Name:             SampleAccountGroupName,
	Description:      nil,
	CommissionPolicy: constants.CommissionPolicyApplied,
	FreightPolicy:    constants.FreightPolicyBilled,
	Type:             constants.AccountGroupTypeTypeGroup,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountGroup)
}
