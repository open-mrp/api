package httpgroup

import (
	"fmt"

	consumptionep "github.com/augno/api/services/api-gateway/endpoints/consumptions"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ConsumptionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ConsumptionsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ConsumptionsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("consumptions endpoint group: core client is required")
	}
	return nil
}

func (*ConsumptionsEndpointGroup) Materialize(config *ConsumptionsEndpointGroupConfig) *ConsumptionsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	consumptionSvc := consumptionep.NewConsumptionSvc(&consumptionep.ConsumptionSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Consumptions Management",
		Description:  "Manage consumptions within production steps.",
		ResourceType: &apiresource.Consumption{},
	}

	getConsumptionEndpoint := (&consumptionep.GetConsumptionEndpoint{}).Materialize().WithService(inner, consumptionSvc)
	createConsumptionEndpoint := (&consumptionep.CreateConsumptionEndpoint{}).Materialize().WithService(inner, consumptionSvc)
	updateConsumptionEndpoint := (&consumptionep.UpdateConsumptionEndpoint{}).Materialize().WithService(inner, consumptionSvc)
	deleteConsumptionEndpoint := (&consumptionep.DeleteConsumptionEndpoint{}).Materialize().WithService(inner, consumptionSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getConsumptionEndpoint,
		createConsumptionEndpoint,
		updateConsumptionEndpoint,
		deleteConsumptionEndpoint,
	}

	return &ConsumptionsEndpointGroup{inner}
}
