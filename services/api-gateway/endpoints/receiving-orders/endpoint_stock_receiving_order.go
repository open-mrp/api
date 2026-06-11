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
	// Per-line stocking details: storage allocations, optional lot number, and any rejected quantity.
	//
	// Lines not listed here are still marked as stocked, but produce no inventory receipts.
	LineItems []StockLineItemRequest `json:"line_items"`
}

// Stocking details for one receiving order line.
type StockLineItemRequest struct {
	// ID of the receiving order line being stocked.
	ReceivingOrderLineID string `json:"receiving_order_line_id"`
	// Lot number to record for the received inventory.
	//
	// A lot is created for the line's item if one with this number does not already exist. Applies to every allocation and any rejected quantity on this line item.
	LotNumber field.Optional[string] `json:"lot_number,omitzero"`
	// Quantity rejected on inspection, as a decimal string.
	//
	// Rejected quantity is recorded on the delivery but is not stocked into inventory.
	RejectedQuantity field.Optional[string] `json:"rejected_quantity,omitzero"`
	// Storage allocations for the accepted quantity.
	//
	// Each allocation creates an inventory receipt for the given quantity at the given location.
	Allocations []AllocationRequest `json:"allocations"`
}

// A portion of a line's accepted quantity placed at a storage location.
type AllocationRequest struct {
	// ID of the storage location to put the quantity away at.
	//
	// When omitted, the inventory receipt is created without a storage location.
	LocationID field.Optional[string] `json:"location_id,omitzero"`
	// Quantity to allocate, as a decimal string.
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

// Stocks the received quantities on a receiving order into inventory.
//
// Every unstocked line with a non-zero quantity is marked as stocked. For each entry in `line_items`, the accepted allocations create inventory receipts at the given storage locations (and lot, if provided), and any `rejected_quantity` is recorded as rejected without entering inventory. A delivery record is created for the stocking event.
//
// If a line was received short of its ordered quantity, a new unstocked line is created automatically for the remainder. Once every line is stocked, the order is marked complete and the originating purchase order is marked fulfilled.
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
