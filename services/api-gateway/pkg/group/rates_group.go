package httpgroup

import (
	"fmt"

	rateep "github.com/augno/api/services/api-gateway/endpoints/rates"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type RatesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RatesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *RatesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("rates endpoint group: core client is required")
	}
	return nil
}

func (*RatesEndpointGroup) Materialize(config *RatesEndpointGroupConfig) *RatesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	rateSvc := rateep.NewRateSvc(&rateep.RateSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Rates Management",
		Description:  "Manage rate records.",
		ResourceType: &apiresource.Rate{},
	}

	updateRateEndpoint := apiendpoint.From(&rateep.UpdateRateEndpoint{}).WithService(inner, rateSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		updateRateEndpoint,
	}

	return &RatesEndpointGroup{inner}
}
