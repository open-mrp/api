package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteCarrierRequest is the request to delete a carrier.
type DeleteCarrierRequest struct {
	// The ID of the carrier to delete.
	CarrierID string `path:"id" validate:"required"`
}

type DeleteCarrierEndpoint struct{}

func (e *DeleteCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteCarrierRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteCarrierRequest, *apiresource.EmptyResource]{
		Title:             "Delete Carrier",
		Description:       "Deletes a carrier and cascades to remove all its options. If the carrier is managed by Shippo, the Shippo account is deactivated.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/carriers/{id}",
		ContentType:       "application/json",
		Request:           &DeleteCarrierRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(CarrierSvc).DeleteCarrier
		},
	}
}
