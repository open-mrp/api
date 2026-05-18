package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update an account group.
type UpdateAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Description.
	Description *string `json:"description,omitempty" nullable:"true"`
	// Commission policy.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
	// Freight policy.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" nullable:"false"`
}

var sampleUpdateAccountGroupRequest = &UpdateAccountGroupRequest{
	Name: new("Updated Wholesale Customers"),
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).UpdateAccountGroup
		},
	})
}
