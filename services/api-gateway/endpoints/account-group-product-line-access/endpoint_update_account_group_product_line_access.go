package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateAccountGroupProductLineAccessRequest is the request to update product line access for an account group.
type UpdateAccountGroupProductLineAccessRequest struct {
	// The ID of the account group.
	AccountGroupID string `path:"account_group_id" validate:"required"`
	// The IDs of the product lines to grant access to.
	ProductLineIDs *[]string `json:"product_line_ids,omitempty" nullable:"false"`
}

var sampleUpdateAccountGroupProductLineAccessRequest = &UpdateAccountGroupProductLineAccessRequest{
	ProductLineIDs: &[]string{apiresource.SampleProductLineID},
}

func (*UpdateAccountGroupProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountGroupProductLineAccessRequest)
}

type UpdateAccountGroupProductLineAccessEndpoint struct{}

func (e *UpdateAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return &apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Update Account Group Product Line Access",
		Description:       "Replaces all product line access for an account group.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		Request:           &UpdateAccountGroupProductLineAccessRequest{},
		Response:          &apiresource.AccountGroupProductLineAccess{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).UpdateAccountGroupProductLineAccess
		},
	}
}
