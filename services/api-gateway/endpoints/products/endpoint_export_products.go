package productep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apierror "github.com/augno/api/shared/errors"
)

// ExportProductsRequest is the request to export products as an Excel file.
type ExportProductsRequest struct {
	// Optional search query.
	Query *string `query:"q"`
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter by product line IDs.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Start of creation date range.
	StartDate *time.Time `query:"start_date"`
	// End of creation date range.
	EndDate *time.Time `query:"end_date"`
}

// Exports all matching products as an Excel file.
type ExportProductsEndpoint struct{}

func (e *ExportProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportProductsRequest, *httptransport.FileDownload]{
		Title:             "Export Products",
		Method:            http.MethodGet,
		ContentType:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:             "/v1/catalog/products/actions/export",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(ProductSvc).ExportProducts
		},
	})
}
