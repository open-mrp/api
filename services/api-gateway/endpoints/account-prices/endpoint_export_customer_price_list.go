package accountpriceep

import (
	"context"
	"net/http"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export a customer's price list.
type ExportCustomerPriceListRequest struct {
	// ID of the customer whose prices are listed.
	CustomerID string `query:"customer_id" validate:"required"`
}

// Downloads a customer's price list as a PDF.
//
// The document covers every product the customer may order, grouped by product line and then by the SKUs that share a price, with the attributes that vary shown as columns. Prices are calculated by the same engine that prices a sales order, so they include the customer's contracted prices and any volume discount they qualify for; a volume break becomes its own price column only where it actually changes a price.
type ExportCustomerPriceListEndpoint struct{}

func (e *ExportCustomerPriceListEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportCustomerPriceListRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportCustomerPriceListRequest, *httptransport.FileDownload]{
		Title:             "Export Price List",
		Method:            http.MethodGet,
		ContentType:       "application/pdf",
		Route:             "/v1/sales/account-prices/actions/export-price-list",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportCustomerPriceListRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(AccountPriceSvc).ExportCustomerPriceList
		},
	})
}
