package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list product line access records, one per account group.
type ListAccountGroupProductLineAccessRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of product line access records, one per account group.
//
// Only account groups that have been granted at least one product line appear. The `q` search term is matched against the account group name.
type ListAccountGroupProductLineAccessEndpoint struct{}

func (e *ListAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountGroupProductLineAccessRequest, *apiresource.List[apiresource.AccountGroupProductLineAccess]] {
	return (&apiendpoint.APIEndpoint[*ListAccountGroupProductLineAccessRequest, *apiresource.List[apiresource.AccountGroupProductLineAccess]]{
		Title:             "List Account Group Product Line Access",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroupProductLineAccess,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupProductLineAccessRequest) (*apiresource.List[apiresource.AccountGroupProductLineAccess], *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).ListAccountGroupProductLineAccess
		},
	})
}
