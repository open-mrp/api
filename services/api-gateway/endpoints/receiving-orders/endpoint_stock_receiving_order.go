package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to stock a receiving order.
type StockReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
	// Line items to stock with allocation details.
	LineItems []StockLineItemRequest `json:"line_items"`
}

// Line item in a stocking request.
type StockLineItemRequest struct {
	// Receiving order line ID.
	ReceivingOrderLineID string `json:"receiving_order_line_id"`
	// Lot number to assign.
	LotNumber *string `json:"lot_number,omitempty"`
	// Rejected quantity value.
	RejectedQuantity *string `json:"rejected_quantity,omitempty"`
	// Storage allocations for this line item.
	Allocations []AllocationRequest `json:"allocations"`
}

// Storage allocation.
type AllocationRequest struct {
	// Location ID to allocate to.
	LocationID *string `json:"location_id,omitempty"`
	// Quantity to allocate.
	Quantity string `json:"quantity"`
}

var sampleStockLocationID = apiresource.SampleLocationID
var sampleStockReceivingOrderRequest = &StockReceivingOrderRequest{
	LineItems: []StockLineItemRequest{
		{
			ReceivingOrderLineID: apiresource.SampleReceivingOrderLineID,
			Allocations: []AllocationRequest{
				{
					LocationID: &sampleStockLocationID,
					Quantity:   "100",
				},
			},
		},
	},
}

func (*StockReceivingOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleStockReceivingOrderRequest)
}

// Stocks a receiving order by allocating line items to storage locations.
type StockReceivingOrderEndpoint struct{}

func (e *StockReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*StockReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return (&apiendpoint.APIEndpoint[*StockReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Stock Receiving Order",
		Method:            http.MethodPost,
		Route:             "/v1/operations/receiving-orders/{id}/actions/stock",
		ContentType:       "application/json",
		Request:           &StockReceivingOrderRequest{},
		Response:          &apiresource.ReceivingOrder{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *StockReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).StockReceivingOrder
		},
	}).WithDocSource(e)
}
