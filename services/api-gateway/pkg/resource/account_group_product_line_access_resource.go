package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

// The set of product lines that accounts in an account group are allowed to order from.
//
// A customer reaches these product lines when the group is its type group or one of its pricing groups. Group access is additive with the customer's own direct access — a customer can order anything granted by either route.
type AccountGroupProductLineAccess struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group_product_line_access"`
	// The account group this access record belongs to.
	//
	// There is at most one access record per account group, so this also identifies the record.
	AccountGroup *AccountGroup `json:"account_group" validate:"required"`
	// Product lines accessible to this account group.
	//
	// Only product lines your account owns can be granted; the shared system product lines never appear here.
	ProductLines *List[ProductLine] `json:"product_lines" validate:"required"`
	// When the account group was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the account group was last updated.
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
	ProductLines: NewList([]ProductLine{*SampleProductLine}, PageInfo{}),
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountGroupProductLineAccess) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountGroupProductLineAccess)
}
