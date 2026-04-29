package productep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListProductsRequest is the request to list products.
type ListProductsRequest struct {
	apiresource.PaginationRequest
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by product line IDs.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Start of creation date range.
	StartDate *time.Time `query:"start_date"`
	// End of creation date range.
	EndDate *time.Time `query:"end_date"`
	// Filter by portal ready status.
	IsPortalReady *bool `query:"is_portal_ready"`
}

type ListProductsEndpoint struct{}

func (e *ListProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductsRequest, *apiresource.List[apiresource.Product]] {
	return &apiendpoint.APIEndpoint[*ListProductsRequest, *apiresource.List[apiresource.Product]]{
		Title:             "List Products",
		Description:       "Returns a paginated list of products for the target account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/products",
		Request:           &ListProductsRequest{},
		Response:          &apiresource.List[apiresource.Product]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductsRequest) (*apiresource.List[apiresource.Product], *apierror.APIError) {
			return svc.(ProductSvc).ListProducts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}
}
