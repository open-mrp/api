package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// AccountGroupProductLineAccess represents the product lines accessible to an account group.
type AccountGroupProductLineAccess struct {
	// The account group.
	AccountGroup *AccountGroup `json:"account_group" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group_product_line_access"`
	// The product lines accessible to this account group.
	ProductLines *List[ProductLine] `json:"product_lines" validate:"required"`
	// When this record was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this record was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountGroupProductLineAccess = &AccountGroupProductLineAccess{
	AccountGroup: &AccountGroup{
		ID:               SampleAccountGroupID,
		Object:           constants.ObjectTypeAccountGroup,
		Name:             SampleAccountGroupName,
		CommissionPolicy: constants.CommissionPolicyApplied,
		FreightPolicy:    constants.FreightPolicyBilled,
		Type:             constants.AccountGroupTypePricingGroup,
		CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	},
	Object: constants.ObjectTypeAccountGroupProductLineAccess,
	ProductLines: NewList([]ProductLine{
		{
			ID:     SampleProductLineID,
			Object: constants.ObjectTypeProductLine,
			Name:   SampleProductLineName,
		},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountGroupProductLineAccess) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountGroupProductLineAccess)
}
