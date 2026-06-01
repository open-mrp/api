package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateAccountGroupProductLineAccessRequest is a request to update product line access for an account group.
type UpdateAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
	// Product line IDs to grant access to.
	ProductLineIDs *[]string `json:"product_line_ids,omitempty"`
}

var sampleUpdateAccountGroupProductLineAccessRequest = &UpdateAccountGroupProductLineAccessRequest{
	ProductLineIDs: &[]string{apiresource.SampleProductLineID},
}

func (*UpdateAccountGroupProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountGroupProductLineAccessRequest)
}

// Replaces all product line access for an account group.
type UpdateAccountGroupProductLineAccessEndpoint struct{}

func (e *UpdateAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Update Account Group Product Line Access",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroupProductLineAccess,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).UpdateAccountGroupProductLineAccess
		},
	})
}
