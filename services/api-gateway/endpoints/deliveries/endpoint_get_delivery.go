package deliveryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a delivery.
type GetDeliveryRequest struct {
	// Delivery ID.
	DeliveryID string `path:"id" validate:"required"`
}

type GetDeliveryEndpoint struct{}

func (e *GetDeliveryEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetDeliveryRequest, *apiresource.Delivery] {
	return &apiendpoint.APIEndpoint[*GetDeliveryRequest, *apiresource.Delivery]{
		Title:             "Get Delivery",
		Description:       "Returns a delivery by ID, including all delivery lines.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/deliveries/{id}",
		Request:           &GetDeliveryRequest{},
		Response:          &apiresource.Delivery{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetDeliveryRequest) (*apiresource.Delivery, *apierror.APIError) {
			return svc.(DeliverySvc).GetDelivery
		},
	}
}
