package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountGroupID = "acgp_01jm4r6700f8nwq3v5hx2d9ktp"
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
