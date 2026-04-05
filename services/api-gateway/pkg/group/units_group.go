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

	listUnitsEndpoint := (&unitep.ListUnitsEndpoint{}).Materialize().WithService(inner, unitSvc)
	getUnitEndpoint := (&unitep.GetUnitEndpoint{}).Materialize().WithService(inner, unitSvc)
	createUnitEndpoint := (&unitep.CreateUnitEndpoint{}).Materialize().WithService(inner, unitSvc)
	updateUnitEndpoint := (&unitep.UpdateUnitEndpoint{}).Materialize().WithService(inner, unitSvc)
	deleteUnitEndpoint := (&unitep.DeleteUnitEndpoint{}).Materialize().WithService(inner, unitSvc)
	validateUnitsEndpoint := (&unitep.ValidateUnitsEndpoint{}).Materialize().WithService(inner, unitSvc)

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
