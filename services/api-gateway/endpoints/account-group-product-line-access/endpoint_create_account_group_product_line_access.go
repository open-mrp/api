package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create product line access for an account group.
type CreateAccountGroupProductLineAccessRequest struct {
	// ID of the account group to grant product line access to.
	AccountGroupID string `json:"account_group_id" validate:"required"`
	// IDs of the product lines the account group is granted access to.
	//
	// Must contain at least one ID, and each one must be a product line your account owns; the shared system product lines cannot be granted.
	ProductLineIDs []string `json:"product_line_ids" validate:"required"`
}

var sampleCreateAccountGroupProductLineAccessRequest = &CreateAccountGroupProductLineAccessRequest{
	AccountGroupID: apiresource.SampleAccountGroupID,
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*CreateAccountGroupProductLineAccessRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountGroupProductLineAccessRequest)
}

// Grants an account group access to a set of product lines.
//
// Every customer that has this group as its type group or as one of its pricing groups can then browse and order those product lines, on top of anything granted to the customer directly.
//
// Each account group can have at most one access record; creating one for an account group that already has one returns a conflict error. Use Update Account Group Product Line Access to change an existing record.
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
		ObjectType:        constants.ObjectTypeAccountGroupProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).CreateAccountGroupProductLineAccess
		},
		LocationFunc: func(resp *apiresource.AccountGroupProductLineAccess) string {
			return "/v1/sales/product-line-access/account-groups/" + resp.AccountGroup.ID
		},
	})
}
