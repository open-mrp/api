package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a service level.
type DeleteServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
}

type DeleteServiceLevelEndpoint struct{}

func (e *DeleteServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *apiresource.EmptyResource]{
		Title:             "Delete Service Level",
		Description:       "Permanently deletes a service level. Fails if the service level is a default (system-synced) level.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		ContentType:       "application/json",
		Request:           &DeleteServiceLevelRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ServiceLevelSvc).DeleteServiceLevel
		},
	}
}
