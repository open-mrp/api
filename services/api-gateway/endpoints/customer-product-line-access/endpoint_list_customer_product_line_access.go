package customerproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list product line access records, one per customer.
type ListCustomerProductLineAccessRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of direct product line access records, one per customer.
//
// Only customers granted at least one product line directly appear; access inherited through a type group or pricing group is not listed here. The `q` search term is matched against the customer name and customer number.
type ListCustomerProductLineAccessEndpoint struct{}

func (e *ListCustomerProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerProductLineAccessRequest, *apiresource.List[apiresource.CustomerProductLineAccess]] {
	return (&apiendpoint.APIEndpoint[*ListCustomerProductLineAccessRequest, *apiresource.List[apiresource.CustomerProductLineAccess]]{
		Title:             "List Customer Product Line Access",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/customers",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCustomerProductLineAccess,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerProductLineAccessRequest) (*apiresource.List[apiresource.CustomerProductLineAccess], *apierror.APIError) {
			return svc.(CustomerProductLineAccessSvc).ListCustomerProductLineAccess
		},
	})
}
