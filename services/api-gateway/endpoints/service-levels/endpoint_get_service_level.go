package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetServiceLevelRequest is the request to retrieve a single service level.
type GetServiceLevelRequest struct {
	// The ID of the carrier.
	CarrierID string `path:"carrier_id" validate:"required"`
	// The ID of the service level to retrieve.
	ServiceLevelID string `path:"id" validate:"required"`
}

type GetServiceLevelEndpoint struct{}

func (e *GetServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetServiceLevelRequest, *apiresource.ServiceLevel] {
	return &apiendpoint.APIEndpoint[*GetServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Get Service Level",
		Description:       "Returns a single service level by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		Request:           &GetServiceLevelRequest{},
		Response:          &apiresource.ServiceLevel{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).GetServiceLevel
		},
	}
}
