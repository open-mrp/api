package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
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
	LotNumber field.Optional[string] `json:"lot_number,omitzero"`
	// Rejected quantity value.
	RejectedQuantity field.Optional[string] `json:"rejected_quantity,omitzero"`
	// Storage allocations for this line item.
	Allocations []AllocationRequest `json:"allocations"`
}

// Storage allocation.
type AllocationRequest struct {
	// Location ID to allocate to.
	LocationID field.Optional[string] `json:"location_id,omitzero"`
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
					LocationID: field.Some(sampleStockLocationID),
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
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *StockReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).StockReceivingOrder
		},
	})
}
