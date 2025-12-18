package healthep

import (
	"context"
	"net/http"
	"sync"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/contracts"
)

type HealthEndpoint struct {
	apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.Healthcheck]

	group    *apiendpoint.APIEndpointGroup
	service  HealthCtrl
	bindOnce sync.Once
	handler  http.HandlerFunc
}

func (e *HealthEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.Healthcheck]{
		Title:             "Get Health Check",
		Description:       "Returns the current health status, environment, and version.",
		Method:            http.MethodGet,
		Route:             "/healthz",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          &apiresource.Healthcheck{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          false,
		Handler: func(svc any) apiendpoint.HandlerFunc[
			*apiresource.EmptyResource, *apiresource.Healthcheck,
		] {
			return apiendpoint.HandlerFunc[
				*apiresource.EmptyResource, *apiresource.Healthcheck,
			](func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *apierror.APIError) {
				return svc.(HealthCtrl).GetHealth(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *HealthEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *HealthEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service HealthCtrl) *HealthEndpoint {
	e.group = g
	e.service = service
	return e
}
