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

// UpdateAccountGroupRequest is the request to partially update an account group.
type UpdateAccountGroupRequest struct {
	// The ID of the account group to update.
	AccountGroupID string `path:"id" validate:"required"`
	// The display name of the account group.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// An optional description of the account group.
	Description *string `json:"description,omitempty" nullable:"true"`
	// The commission status code.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
	// The freight status code.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" nullable:"false"`
}

var sampleUpdateAccountGroupRequest = &UpdateAccountGroupRequest{
	Name: new("Updated Wholesale Customers"),
}

func (*UpdateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountGroupRequest)
}

type UpdateAccountGroupEndpoint struct{}

func (e *UpdateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup] {
	return &apiendpoint.APIEndpoint[*UpdateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Update Account Group",
		Description:       "Partially updates an account group.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/account-groups/{id}",
		ContentType:       "application/json",
		Request:           &UpdateAccountGroupRequest{},
		Response:          &apiresource.AccountGroup{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).UpdateAccountGroup
		},
	}
}
