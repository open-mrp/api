package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountGroupID = "acgp_018e88072d1320808dc979cfac"
const SampleAccountGroupName = "Wholesale Customers"

// A named grouping of customer accounts, used for pricing rules or to categorize accounts.
type AccountGroup struct {
	// Account group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group"`
	// How this account group is used.
	//
	// - `pricing_group`: used for pricing rules, such as a "Preferred" group that receives a special discount.
	// - `type_group`: used to categorize accounts, such as "Consumers" or "Distributors".
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// Display name of the account group.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// Free-form description of the account group.
	Description *string `json:"description"`
	// How sales commission applies to accounts in this group.
	//
	// - `commission_applied`: sales commission is calculated on orders from accounts in this group.
	// - `commission_exempt`: orders from accounts in this group are exempt from commission.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// How freight charges apply to orders from accounts in this group.
	//
	// - `free_freight`: customers within this group will not have to pay for freight.
	// - `billed_freight`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountGroup = &AccountGroup{
	ID:               SampleAccountGroupID,
	Object:           constants.ObjectTypeAccountGroup,
	Type:             constants.AccountGroupTypeTypeGroup,
	Name:             SampleAccountGroupName,
	Description:      nil,
	CommissionPolicy: constants.CommissionPolicyApplied,
	FreightPolicy:    constants.FreightPolicyBilled,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountGroup)
}
