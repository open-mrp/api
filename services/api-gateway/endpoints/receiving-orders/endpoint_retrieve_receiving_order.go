package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a receiving order.
type RetrieveReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
}

// Returns a receiving order by ID.
type RetrieveReceivingOrderEndpoint struct{}

func (e *RetrieveReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return (&apiendpoint.APIEndpoint[*RetrieveReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Retrieve Receiving Order",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).GetReceivingOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeReceivingOrder,
			// The item's unit group is offered because the receiving screens measure against it: a line is checked off in the unit it was ordered in, and the stocking dialog offers that group's units to put it away in.
			Fields: []string{"supplier", "totals", "related", "related.purchase_order", "related.deliveries", "lines", "lines.item", "lines.order_line", "lines.order_line.item", "lines.order_line.quantity_ordered", "lines.order_line.quantity_ordered.unit", "lines.order_line.unit_price", "lines.order_line.unit_price.numerator_unit", "lines.order_line.unit_price.denominator_unit", "lines.quantity", "lines.quantity.unit", "lines.quantity_ordered", "lines.quantity_ordered.unit", "lines.item.category", "lines.item.category.unit_group", "lines.item.category.unit_group.base_unit", "lines.item.category.unit_group.associated_units", "lines.item.category.unit_group.associated_units.unit"},
		}),
	})
}
