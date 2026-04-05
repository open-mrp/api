package httpgroup

import (
	healthep "github.com/augno/api/services/api-gateway/endpoints/health"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type HealthEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type HealthEndpointGroupConfig struct {
}

func (*HealthEndpointGroup) Materialize(config HealthEndpointGroupConfig) *HealthEndpointGroup {
	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Health",
		Description:  "Health monitoring endpoints for service status and environment information.",
		ResourceType: &apiresource.Healthcheck{},
	}

	healthController := healthep.NewHealthSvc()
	healthEndpoint := (&healthep.HealthEndpoint{}).Materialize().WithService(inner, healthController)
	inner.Endpoints = []apiendpoint.APIEndpointer{healthEndpoint}

	return &HealthEndpointGroup{inner}
}
