package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
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
		ObjectType:        constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).GetReceivingOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeReceivingOrder,
			Fields:     []string{"supplier", "purchase_order", "lines", "lines.order_line"},
		}),
	})
}
