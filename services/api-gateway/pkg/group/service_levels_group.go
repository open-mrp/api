package httpgroup

import (
	"fmt"

	servicelevelep "github.com/augno/api/services/api-gateway/endpoints/service-levels"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ServiceLevelsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ServiceLevelsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ServiceLevelsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("service levels endpoint group: core client is required")
	}
	return nil
}

func (*ServiceLevelsEndpointGroup) Materialize(config *ServiceLevelsEndpointGroupConfig) *ServiceLevelsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	serviceLevelSvc := servicelevelep.NewServiceLevelSvc(&servicelevelep.ServiceLevelSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Service Levels Management",
		Description:  "List and manage service levels (shipping service levels).",
		ResourceType: &apiresource.ServiceLevel{},
	}

	listServiceLevelsEndpoint := (&servicelevelep.ListServiceLevelsEndpoint{}).Materialize().WithService(inner, serviceLevelSvc)
	getServiceLevelEndpoint := (&servicelevelep.GetServiceLevelEndpoint{}).Materialize().WithService(inner, serviceLevelSvc)
	createServiceLevelEndpoint := (&servicelevelep.CreateServiceLevelEndpoint{}).Materialize().WithService(inner, serviceLevelSvc)
	updateServiceLevelEndpoint := (&servicelevelep.UpdateServiceLevelEndpoint{}).Materialize().WithService(inner, serviceLevelSvc)
	deleteServiceLevelEndpoint := (&servicelevelep.DeleteServiceLevelEndpoint{}).Materialize().WithService(inner, serviceLevelSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listServiceLevelsEndpoint,
		getServiceLevelEndpoint,
		createServiceLevelEndpoint,
		updateServiceLevelEndpoint,
		deleteServiceLevelEndpoint,
	}

	return &ServiceLevelsEndpointGroup{inner}
}
