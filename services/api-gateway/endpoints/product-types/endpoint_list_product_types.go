package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListProductTypesRequest is the request to list product types with optional filters.
type ListProductTypesRequest struct {
	apiresource.PaginationRequest
}

type ListProductTypesEndpoint struct{}

func (e *ListProductTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]] {
	return &apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]]{
		Title:             "List Product Types",
		Description:       "Returns a paginated list of product types. Product types are global and not scoped to a specific account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-types",
		Request:           &ListProductTypesRequest{},
		Response:          &apiresource.List[apiresource.ProductType]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError) {
			return svc.(ProductTypeSvc).ListProductTypes
		},
	}
}
