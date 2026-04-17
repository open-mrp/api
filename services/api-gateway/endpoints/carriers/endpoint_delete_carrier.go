package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a carrier.
type DeleteCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

type DeleteCarrierEndpoint struct{}

func (e *DeleteCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCarrierRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteCarrierRequest, *apiresource.EmptyResource]{
		Title:             "Delete Carrier",
		Description:       "Deletes a carrier and cascades to remove all options. If the carrier is managed by Shippo, the Shippo account is deactivated.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/carriers/{id}",
		ContentType:       "application/json",
		Request:           &DeleteCarrierRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CarrierSvc).DeleteCarrier
		},
	}
}
