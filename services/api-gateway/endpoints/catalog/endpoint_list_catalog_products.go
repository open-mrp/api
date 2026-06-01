package catalogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list products in a catalog product line.
type ListCatalogProductsRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns a paginated list of products in a specific product line, grouped by item category.
type ListCatalogProductsEndpoint struct{}

func (e *ListCatalogProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCatalogProductsRequest, *apiresource.List[apiresource.CatalogCategory]] {
	return (&apiendpoint.APIEndpoint[*ListCatalogProductsRequest, *apiresource.List[apiresource.CatalogCategory]]{
		Title:             "List Catalog Products",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/catalog/product-lines/{id}/products",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCatalogCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCatalogProductsRequest) (*apiresource.List[apiresource.CatalogCategory], *apierror.APIError) {
			return svc.(CatalogSvc).ListCatalogProducts
		},
	})
}
