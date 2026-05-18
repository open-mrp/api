package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateAccountGroupProductLineAccessRequest is a request to create product line access for an account group.
type CreateAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `json:"account_group_id" validate:"required"`
	// Product line IDs to grant access to.
	ProductLineIDs []string `json:"product_line_ids" validate:"required"`
}

var sampleCreateAccountGroupProductLineAccessRequest = &CreateAccountGroupProductLineAccessRequest{
	AccountGroupID: apiresource.SampleAccountGroupID,
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*CreateAccountGroupProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountGroupProductLineAccessRequest)
}

// Creates product line access for an account group.
type CreateAccountGroupProductLineAccessEndpoint struct{}

func (e *CreateAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*CreateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Create Account Group Product Line Access",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).CreateAccountGroupProductLineAccess
		},
		LocationFunc: func(resp *apiresource.AccountGroupProductLineAccess) string {
			return "/v1/sales/product-line-access/account-groups/" + resp.AccountGroup.ID
		},
	})
}
