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
		Description:  "List and manage units.",
		ResourceType: &apiresource.Unit{},
	}

	listUnitsEndpoint := apiendpoint.From(&unitep.ListUnitsEndpoint{}).WithService(inner, unitSvc)
	getUnitEndpoint := apiendpoint.From(&unitep.RetrieveUnitEndpoint{}).WithService(inner, unitSvc)
	createUnitEndpoint := apiendpoint.From(&unitep.CreateUnitEndpoint{}).WithService(inner, unitSvc)
	updateUnitEndpoint := apiendpoint.From(&unitep.UpdateUnitEndpoint{}).WithService(inner, unitSvc)
	deleteUnitEndpoint := apiendpoint.From(&unitep.DeleteUnitEndpoint{}).WithService(inner, unitSvc)
	validateUnitsEndpoint := apiendpoint.From(&unitep.ValidateUnitsEndpoint{}).WithService(inner, unitSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listUnitsEndpoint,
		getUnitEndpoint,
		createUnitEndpoint,
		updateUnitEndpoint,
		deleteUnitEndpoint,
		validateUnitsEndpoint,
	}

	return &UnitsEndpointGroup{inner}
}
