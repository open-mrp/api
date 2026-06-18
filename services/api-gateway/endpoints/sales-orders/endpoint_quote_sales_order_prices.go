package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to quote sales-order line prices without creating an order.
type QuoteSalesOrderPricesRequest struct {
	// ID of the customer account the prices are for.
	BuyerAccountID string `json:"buyer_account_id" validate:"required"`
	// Lines to price.
	Lines []QuoteSalesOrderLineInput `json:"lines" validate:"required,min=1,dive"`
}

// A line to price in a quote request.
type QuoteSalesOrderLineInput struct {
	// ID of the product to price.
	ProductID string `json:"product_id" validate:"required"`
	// Quantity ordered. The unit must belong to the product's unit group.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
}

// Quoted unit prices for the requested lines, in request order.
type QuoteSalesOrderPricesResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_price_quote"`
	// Priced lines, in the same order as the request.
	Lines []QuotedSalesOrderLine `json:"lines"`
}

// One priced line in a quote response.
type QuotedSalesOrderLine struct {
	// ID of the product priced.
	ProductID string `json:"product_id" validate:"required"`
	// Calculated unit price, as a decimal string.
	UnitPriceValue string `json:"unit_price_value" format:"decimal"`
	// Unit ID for the unit price's numerator (the currency).
	UnitPriceNumeratorUnitID string `json:"unit_price_numerator_unit_id"`
	// Unit ID for the unit price's denominator (the unit being sold).
	UnitPriceDenominatorUnitID string `json:"unit_price_denominator_unit_id"`
}

var sampleQuoteSalesOrderPricesRequest = &QuoteSalesOrderPricesRequest{
	BuyerAccountID: apiresource.SampleCustomerID,
	Lines: []QuoteSalesOrderLineInput{
		{
			ProductID: apiresource.SampleProductID,
			Quantity: apirequest.QuantityInput{
				Value:  "10",
				UnitID: apiresource.SampleUnitID,
			},
		},
	},
}

func (*QuoteSalesOrderPricesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleQuoteSalesOrderPricesRequest)
}

var sampleQuoteSalesOrderPricesResponse = &QuoteSalesOrderPricesResponse{
	Object: constants.ObjectTypeSalesOrderPriceQuote,
	Lines: []QuotedSalesOrderLine{
		{
			ProductID:                  apiresource.SampleProductID,
			UnitPriceValue:             "25.00",
			UnitPriceNumeratorUnitID:   apiresource.SampleUnitID,
			UnitPriceDenominatorUnitID: apiresource.SampleUnitID,
		},
	},
}

func (*QuoteSalesOrderPricesResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleQuoteSalesOrderPricesResponse)
}

// Calculates the unit price for each line without creating an order.
//
// Use this to display prices to users as they build an order. Prices are computed
// server-side from the product's list price, contracted account prices, and applicable
// discounts — the same logic used when an order is created. Internal price overrides are
// not accepted here; the calculated price is always returned.
type QuoteSalesOrderPricesEndpoint struct{}

func (e *QuoteSalesOrderPricesEndpoint) Materialize() *apiendpoint.APIEndpoint[*QuoteSalesOrderPricesRequest, *QuoteSalesOrderPricesResponse] {
	return (&apiendpoint.APIEndpoint[*QuoteSalesOrderPricesRequest, *QuoteSalesOrderPricesResponse]{
		Title:             "Quote Sales Order Prices",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/price-quote",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *QuoteSalesOrderPricesRequest) (*QuoteSalesOrderPricesResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).QuoteSalesOrderPrices
		},
	})
}
