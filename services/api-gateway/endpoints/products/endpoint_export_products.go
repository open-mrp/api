package productep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export products as an Excel file.
type ExportProductsRequest struct {
	// Free-text search matched against the SKU and description of each product's item.
	Query *string `query:"q"`
	// Filter by the item category the product's item belongs to.
	CategoryIDs []string `query:"category_ids"`
	// Filter to products whose item carries at least one of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter by product line IDs.
	//
	// Combined with `customer_ids`, products matching either filter are exported.
	ProductLineIDs []string `query:"product_line_ids"`
	// Restrict the export to products these customer accounts are entitled to buy.
	//
	// A product matches when its product line has been granted to the customer directly, through the customer's account group, or through the account group used for the customer's pricing.
	CustomerIDs []string `query:"customer_ids"`
	// Start of creation date range.
	StartDate *time.Time `query:"start_date"`
	// End of creation date range.
	EndDate *time.Time `query:"end_date"`
}

// Exports matching products as an Excel workbook.
//
// The response is a file download, not JSON, and it is not paginated: every product matching the filters is written to a single sheet, one row per product, with columns for the product ID, SKU, description, category, product line, and unit price and cost with their units, plus one column for each category property in use. As with the product list, only products of type `sale` are exported.
type ExportProductsEndpoint struct{}

func (e *ExportProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportProductsRequest, *httptransport.FileDownload]{
		Title:               "Export Products",
		Method:              http.MethodGet,
		ContentType:         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:               "/v1/catalog/products/actions/export",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(ProductSvc).ExportProducts
		},
	})
}
