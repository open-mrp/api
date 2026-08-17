package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to re-quote an order's freight charge.
type QuoteSalesOrderFreightRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// The freshly estimated freight charge for a sales order.
type QuoteSalesOrderFreightResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_freight_quote"`
	// Estimated freight unit price.
	UnitPrice *apiresource.ComputedRate `json:"unit_price"`
}

var sampleQuoteSalesOrderFreightResponse = &QuoteSalesOrderFreightResponse{
	Object:    constants.ObjectTypeSalesOrderFreightQuote,
	UnitPrice: apiresource.NewComputedRate("24.50", apiresource.SampleCurrencyUnit, apiresource.SampleEachUnit),
}

func (*QuoteSalesOrderFreightResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleQuoteSalesOrderFreightResponse)
}

// Re-estimates the freight (shipping) charge for an order using the latest carrier rates.
//
// Computes what the order's freight charge would be from its current ship-to address, carrier, service level, and line items — applying the same freight-exemption, flat-rate, and live carrier-rate logic used when the order is created. The order is not modified: the returned amount is a quote to review, and callers apply it by updating the order's shipping line. Use this to refresh freight after changing the address or line items, or at any time to re-price against current rates.
type QuoteSalesOrderFreightEndpoint struct{}

func (e *QuoteSalesOrderFreightEndpoint) Materialize() *apiendpoint.APIEndpoint[*QuoteSalesOrderFreightRequest, *QuoteSalesOrderFreightResponse] {
	return (&apiendpoint.APIEndpoint[*QuoteSalesOrderFreightRequest, *QuoteSalesOrderFreightResponse]{
		Title:             "Quote Sales Order Freight",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/quote-freight",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *QuoteSalesOrderFreightRequest) (*QuoteSalesOrderFreightResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).QuoteSalesOrderFreight
		},
	})
}
