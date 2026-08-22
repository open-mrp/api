package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list product types.
type ListProductTypesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of product types.
//
// Product types are global and not scoped to a specific account. The `q` search term is matched against the product type name.
type ListProductTypesEndpoint struct{}

func (e *ListProductTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]] {
	return (&apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]]{
		Title:               "List Product Types",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/product-types",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeProductType,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError) {
			return svc.(ProductTypeSvc).ListProductTypes
		},
	})
}
