package healthep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Returns the current health status of the API.
//
// The check is shallow: a successful response confirms the API is running and serving requests, and does not probe the database or any downstream service. It is intended for uptime monitors and load-balancer probes, and is not recorded in the request log.
type HealthEndpoint struct{}

func (e *HealthEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.Healthcheck] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.Healthcheck]{
		Title:             "Get Health Check",
		Method:            http.MethodGet,
		Route:             "/healthz",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *apierror.APIError) {
			return svc.(HealthSvc).GetHealth
		},
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
	})
}
