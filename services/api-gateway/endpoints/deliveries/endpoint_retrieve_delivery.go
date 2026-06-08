package deliveryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a delivery.
type RetrieveDeliveryRequest struct {
	// Delivery ID.
	DeliveryID string `path:"id" validate:"required"`
}

// Returns a delivery by ID, including all delivery lines.
type RetrieveDeliveryEndpoint struct{}

func (e *RetrieveDeliveryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDeliveryRequest, *apiresource.Delivery] {
	return (&apiendpoint.APIEndpoint[*RetrieveDeliveryRequest, *apiresource.Delivery]{
		Title:             "Retrieve Delivery",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/deliveries/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDelivery,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDeliveryRequest) (*apiresource.Delivery, *apierror.APIError) {
			return svc.(DeliverySvc).GetDelivery
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDelivery,
			Fields:     []string{"purchase_order", "lines"},
		}),
	})
}
