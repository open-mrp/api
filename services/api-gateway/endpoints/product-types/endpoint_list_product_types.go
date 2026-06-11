package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list product types.
type ListProductTypesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of product types.
//
// Product types are global and not scoped to a specific account.
type ListProductTypesEndpoint struct{}

func (e *ListProductTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]] {
	return (&apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]]{
		Title:             "List Product Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-types",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductType,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError) {
			return svc.(ProductTypeSvc).ListProductTypes
		},
	})
}
