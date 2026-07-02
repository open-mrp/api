package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// The set of product lines that accounts in an account group are allowed to order from.
type AccountGroupProductLineAccess struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group_product_line_access"`
	// The account group this access record belongs to.
	//
	// There is at most one access record per account group, so this also identifies the record.
	AccountGroup *AccountGroup `json:"account_group" validate:"required"`
	// Product lines accessible to this account group.
	ProductLines *List[ProductLine] `json:"product_lines" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountGroupProductLineAccess = &AccountGroupProductLineAccess{
	Object: constants.ObjectTypeAccountGroupProductLineAccess,
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
