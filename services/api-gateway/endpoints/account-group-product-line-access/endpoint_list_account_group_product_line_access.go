package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list product line access records, one per account group.
type ListAccountGroupProductLineAccessRequest struct {
	apiresource.PaginationRequest
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
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroupProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupProductLineAccessRequest) (*apiresource.List[apiresource.AccountGroupProductLineAccess], *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).ListAccountGroupProductLineAccess
		},
	})
}
