package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountGroupID = "acgp_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleAccountGroupName = "Wholesale Customers"

// AccountGroup represents an account group used for organizing customer accounts.
type AccountGroup struct {
	// The unique identifier for the account group.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group"`
	// The display name of the account group.
	Name string `json:"name" validate:"required"`
	// A description of the account group.
	Description *string `json:"description"`
	// The commission policy of the account group.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// The freight policy of the account group.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// The account group type.
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// When this account group was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this account group was last updated.
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
