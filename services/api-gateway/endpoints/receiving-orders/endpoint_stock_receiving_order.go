package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to stock a receiving order.
type StockReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
	// Per-line stocking details: where to put the goods away, which lot to record them under, and how much was refused on inspection.
	//
	// Unstocked lines left out of this list are still marked as stocked, but nothing is added to inventory for them and they contribute no delivery lines.
	LineItems []StockLineItemRequest `json:"line_items,omitzero"`
}

// Stocking details for one receiving order line.
type StockLineItemRequest struct {
	// ID of the receiving order line being stocked.
	ReceivingOrderLineID string `json:"receiving_order_line_id"`
	// Lot number to record for the received goods.
	//
	// A lot is created for the line's item if one with this number does not already exist for it. The lot applies to every allocation and to any rejected quantity on this line item.
	LotNumber field.Optional[string] `json:"lot_number,omitzero"`
	// Quantity refused on inspection, as a decimal string.
	//
	// The refused quantity is recorded on the delivery and on the receiving order line's `rejected_quantity`, but never enters inventory.
	RejectedQuantity field.Optional[string] `json:"rejected_quantity,omitzero"`
	// Storage allocations for the quantity being accepted.
	//
	// Each allocation creates an inventory receipt for the given quantity at the given location, so a single line can be split across several locations.
	Allocations []AllocationRequest `json:"allocations,omitzero"`
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
// Every unstocked line with a non-zero quantity is marked as stocked. For each entry in `line_items`, the allocations create inventory receipts at the given storage locations (and lot, if one was given), and any `rejected_quantity` is recorded as refused without entering inventory. One delivery is recorded for the whole stocking event, with a line per allocation and a line per refused quantity.
//
// The newly received stock is then applied to any open inventory issues for the same item, oldest first, so demand already waiting on the item is satisfied automatically.
//
// If a line was received short of its ordered quantity, a new unstocked line is created automatically for the remainder. Once every line is stocked, the order is marked complete and the originating purchase order is marked fulfilled.
//
// A receiving order with no unstocked, non-zero lines is returned untouched: no delivery is recorded and no inventory is created.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *StockReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).StockReceivingOrder
		},
	})
}
