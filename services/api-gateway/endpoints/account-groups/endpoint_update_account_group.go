package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an account group.
type UpdateAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Description.
	Description field.Clearable[string] `json:"description,omitzero"`
	// Commission policy.
	//
	// - `commission_exempt`: no commission applies.
	// - `commission_applied`: commission applies; if the account group is within a sales rep's territory, it will be assigned to that rep unless overridden.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Freight policy.
	//
	// - `free_freight`: customers within this group will not have to pay for freight.
	// - `billed_freight`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
}

var sampleUpdateAccountGroupRequest = &UpdateAccountGroupRequest{
	Name: field.Some("Updated Wholesale Customers"),
}

func (*UpdateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountGroupRequest)
}

// Partially updates an account group.
type UpdateAccountGroupEndpoint struct{}

func (e *UpdateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Update Account Group",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).UpdateAccountGroup
		},
	})
}
