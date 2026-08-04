package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list shipments.
type ListShipmentsRequest struct {
	apiresource.PaginationRequest
	// Only include shipments with this status, either `packed` or `shipped`.
	Status *string `query:"status"`
	// Only include shipments containing at least one line for any of these items.
	ItemIDs []string `query:"item_ids"`
	// Only include shipments for any of these customers.
	CustomerIDs []string `query:"customer_ids"`
	// Only include shipments containing at least one line whose product belongs to any of these product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Only include shipments whose customer belongs to any of these customer groups.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Only include shipments whose customer is assigned to any of these sales reps.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Only include shipments created on or after this date (`YYYY-MM-DD`).
	//
	// Filters on when the shipment was created, not on when it was shipped.
	StartDate *string `query:"start_date"`
	// Only include shipments created on or before this date (`YYYY-MM-DD`).
	//
	// Filters on when the shipment was created, not on when it was shipped.
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of shipments, newest first.
//
// Filters combine with AND, while the values within a single list filter combine with OR. The `q` search term matches the shipment number, note, bill of lading and master tracking number, as well as the sales order number, customer name and customer PO number.
type ListShipmentsEndpoint struct{}

func (e *ListShipmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShipmentsRequest, *apiresource.List[apiresource.Shipment]] {
	return (&apiendpoint.APIEndpoint[*ListShipmentsRequest, *apiresource.List[apiresource.Shipment]]{
		Title:               "List Shipments",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/shipments",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShipmentsRequest) (*apiresource.List[apiresource.Shipment], *apierror.APIError) {
			return svc.(ShipmentSvc).ListShipments
		},
		ObjectType: constants.ObjectTypeShipment,
		// The list summary carries customer_id + sales_order_id; expose exactly the
		// includes those FK ids can resolve (loaders are account-scoped, same account).
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"customer", "sales_order", "lines"},
		}),
	})
}
