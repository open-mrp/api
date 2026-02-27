package httpgroup

import (
	"fmt"

	unitep "github.com/augno/api/services/api-gateway/endpoints/units"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type UnitsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type UnitsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *UnitsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("units endpoint group: core client is required")
	}
	return nil
}

func (*UnitsEndpointGroup) Materialize(config *UnitsEndpointGroupConfig) *UnitsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	unitSvc := unitep.NewUnitSvc(&unitep.UnitSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Units Management",
		Description:  "Handles listing and managing units.",
		ResourceType: &apiresource.Unit{},
	}

	listUnitsEndpoint := (&unitep.ListUnitsEndpoint{}).Materialize().WithService(inner, unitSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listUnitsEndpoint,
	}

	return &UnitsEndpointGroup{inner}
}
