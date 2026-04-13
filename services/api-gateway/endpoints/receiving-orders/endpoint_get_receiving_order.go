package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a receiving order.
type GetReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
}

type GetReceivingOrderEndpoint struct{}

func (e *GetReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return &apiendpoint.APIEndpoint[*GetReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Get Receiving Order",
		Description:       "Returns a receiving order by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{id}",
		Request:           &GetReceivingOrderRequest{},
		Response:          &apiresource.ReceivingOrder{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).GetReceivingOrder
		},
	}
}
