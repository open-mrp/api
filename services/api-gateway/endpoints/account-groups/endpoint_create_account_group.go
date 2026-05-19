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

// Request to create an account group.
type CreateAccountGroupRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Account group type.
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// Commission policy.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" default:"commission_exempt"`
	// Freight policy.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" default:"billed_freight"`
	// Description.
	Description *string `json:"description,omitempty"`
}

var sampleCreateAccountGroupRequest = &CreateAccountGroupRequest{
	Name: "Wholesale Customers",
	Type: constants.AccountGroupTypeTypeGroup,
}

func (*CreateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountGroupRequest)
}

// Creates an account group.
type CreateAccountGroupEndpoint struct{}

func (e *CreateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup] {
	return (&apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Create Account Group",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-groups",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).CreateAccountGroup
		},
		LocationFunc: func(resp *apiresource.AccountGroup) string {
			return "/v1/sales/account-groups/" + resp.ID
		},
	})
}
