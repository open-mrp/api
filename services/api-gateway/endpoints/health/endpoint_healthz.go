package healthep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type HealthEndpoint struct{}

func (e *HealthEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.Healthcheck] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.Healthcheck]{
		Title:             "Get Health Check",
		Description:       "Returns the current health status of the API.",
		Method:            http.MethodGet,
		Route:             "/healthz",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          &apiresource.Healthcheck{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *apierror.APIError) {
			return svc.(HealthSvc).GetHealth
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
			SkipRequestLogging:     true,
		},
	}
}
