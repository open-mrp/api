package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteServiceLevelRequest is the request to delete a service level.
type DeleteServiceLevelRequest struct {
	// The ID of the carrier.
	CarrierID string `path:"carrier_id" validate:"required"`
	// The ID of the service level to delete.
	ServiceLevelID string `path:"id" validate:"required"`
}

type DeleteServiceLevelEndpoint struct{}

func (e *DeleteServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *apiresource.EmptyResource]{
		Title:             "Delete Service Level",
		Description:       "Permanently deletes a service level. Default (system-synced) service levels cannot be deleted.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		ContentType:       "application/json",
		Request:           &DeleteServiceLevelRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ServiceLevelSvc).DeleteServiceLevel
		},
	}
}
