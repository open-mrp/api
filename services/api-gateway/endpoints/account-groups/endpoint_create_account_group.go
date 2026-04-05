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

// CreateAccountGroupRequest is the request to create a new account group.
type CreateAccountGroupRequest struct {
	// The display name of the account group.
	Name string `json:"name" validate:"required"`
	// The account group type code.
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// The commission status code.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" default:"commission_exempt" nullable:"false"`
	// The freight status code.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" default:"billed_freight" nullable:"false"`
	// An optional description of the account group.
	Description *string `json:"description,omitempty"`
}

var sampleCreateAccountGroupRequest = &CreateAccountGroupRequest{
	Name: "Wholesale Customers",
	Type: constants.AccountGroupTypeTypeGroup,
}

func (*CreateAccountGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountGroupRequest)
}

type CreateAccountGroupEndpoint struct{}

func (e *CreateAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup] {
	return &apiendpoint.APIEndpoint[*CreateAccountGroupRequest, *apiresource.AccountGroup]{
		Title:             "Create Account Group",
		Description:       "Creates a new account group.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/account-groups",
		Request:           &CreateAccountGroupRequest{},
		Response:          &apiresource.AccountGroup{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
			return svc.(AccountGroupSvc).CreateAccountGroup
		},
	}
}
