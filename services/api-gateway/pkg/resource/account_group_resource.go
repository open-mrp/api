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
	// Description.
	Description *string `json:"description"`
	// Commission policy.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Freight policy.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Account group type.
	//
	// The type `pricing_group` indicates this account group is utilized for pricing rules. For example, you may have a 'Preferred' price group that receives a special discount rate. The type `type_group` indicates the account group is utilized to categorize a set of accounts. For example, you may have a group of accounts that are 'Consumers' or 'Distributors'.
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
