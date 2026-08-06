package productep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which products land in the exported file.
type ExportProductsRequest struct {
	// Free-text search matched against the SKU and description of each product's item.
	Query *string `json:"q"`
	// Filter by the item category the product's item belongs to.
	CategoryIDs []string `json:"category_ids"`
	// Filter to products whose item carries at least one of these attributes.
	AttributeIDs []string `json:"attribute_ids"`
	// Filter by product line IDs.
	//
	// Combined with `customer_ids`, products matching either filter are exported.
	ProductLineIDs []string `json:"product_line_ids"`
	// Restrict the export to products these customer accounts are entitled to buy.
	//
	// A product matches when its product line has been granted to the customer directly, through the customer's account group, or through the account group used for the customer's pricing.
	CustomerIDs []string `json:"customer_ids"`
	// Start of creation date range.
	StartDate *time.Time `json:"starts_at"`
	// End of creation date range.
	EndDate *time.Time `json:"ends_at"`
}

var sampleExportProductsRequest = &ExportProductsRequest{}

func (*ExportProductsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportProductsRequest)
}

// Starts an export of every matching product and returns the job that tracks it; as with the
// product list, only products of type `sale` are exported.
type ExportProductsEndpoint struct{}

func (e *ExportProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportProductsRequest, *apiresource.Job]{
		Title:             "Export Products",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/products/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductSvc).ExportProducts
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
