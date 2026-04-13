package catalogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list catalog product lines.
type ListCatalogProductLinesRequest struct {
	apiresource.PaginationRequest
}

type ListCatalogProductLinesEndpoint struct{}

func (e *ListCatalogProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCatalogProductLinesRequest, *apiresource.List[apiresource.CatalogProductLine]] {
	return &apiendpoint.APIEndpoint[*ListCatalogProductLinesRequest, *apiresource.List[apiresource.CatalogProductLine]]{
		Title:             "List Catalog Product Lines",
		Description:       "Returns a paginated list of product lines available in the catalog. Customers only see product lines they have access to.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/catalog/product-lines",
		ContentType:       "application/json",
		Request:           &ListCatalogProductLinesRequest{},
		Response:          &apiresource.List[apiresource.CatalogProductLine]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCatalogProductLinesRequest) (*apiresource.List[apiresource.CatalogProductLine], *apierror.APIError) {
			return svc.(CatalogSvc).ListCatalogProductLines
		},
	}
}
