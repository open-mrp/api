package deliveryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a delivery.
type RetrieveDeliveryRequest struct {
	// Delivery ID.
	DeliveryID string `path:"id" validate:"required"`
}

type RetrieveDeliveryEndpoint struct{}

func (e *RetrieveDeliveryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDeliveryRequest, *apiresource.Delivery] {
	return &apiendpoint.APIEndpoint[*RetrieveDeliveryRequest, *apiresource.Delivery]{
		Title:             "Retrieve Delivery",
		Description:       "Returns a delivery by ID, including all delivery lines.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/deliveries/{id}",
		Request:           &RetrieveDeliveryRequest{},
		Response:          &apiresource.Delivery{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError) {
			return svc.(DeliverySvc).GetDelivery
		},
	}
}
