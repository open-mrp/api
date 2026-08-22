package httpgroup

import (
	"fmt"

	servicelevelep "github.com/open-mrp/api/services/api-gateway/endpoints/service-levels"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type ServiceLevelsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ServiceLevelsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Service Levels",
		Description:  "List and manage service levels (shipping service levels).",
		ResourceType: &apiresource.ServiceLevel{},
	}

	listServiceLevelsEndpoint := apiendpoint.From(&servicelevelep.ListServiceLevelsEndpoint{}).WithService(inner, serviceLevelSvc)
	getServiceLevelEndpoint := apiendpoint.From(&servicelevelep.RetrieveServiceLevelEndpoint{}).WithService(inner, serviceLevelSvc)
	createServiceLevelEndpoint := apiendpoint.From(&servicelevelep.CreateServiceLevelEndpoint{}).WithService(inner, serviceLevelSvc)
	updateServiceLevelEndpoint := apiendpoint.From(&servicelevelep.UpdateServiceLevelEndpoint{}).WithService(inner, serviceLevelSvc)
	deleteServiceLevelEndpoint := apiendpoint.From(&servicelevelep.DeleteServiceLevelEndpoint{}).WithService(inner, serviceLevelSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listServiceLevelsEndpoint,
		getServiceLevelEndpoint,
		createServiceLevelEndpoint,
		updateServiceLevelEndpoint,
		deleteServiceLevelEndpoint,
	}

	return &ServiceLevelsEndpointGroup{inner}
}
