package httpgroup

import (
	"fmt"

	unitgroupep "github.com/augno/api/services/api-gateway/endpoints/unit-groups"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type UnitGroupsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type UnitGroupsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *UnitGroupsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("unit groups endpoint group: core client is required")
	}
	return nil
}

func (*UnitGroupsEndpointGroup) Materialize(config *UnitGroupsEndpointGroupConfig) *UnitGroupsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	unitGroupSvc := unitgroupep.NewUnitGroupSvc(&unitgroupep.UnitGroupSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Unit Groups Management",
		Description:  "List and manage unit groups and their associated units.",
		ResourceType: &apiresource.UnitGroup{},
	}

	listEndpoint := (&unitgroupep.ListUnitGroupsEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	getEndpoint := (&unitgroupep.GetUnitGroupEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	createEndpoint := (&unitgroupep.CreateUnitGroupEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	updateEndpoint := (&unitgroupep.UpdateUnitGroupEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	deleteEndpoint := (&unitgroupep.DeleteUnitGroupEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	createUnitEndpoint := (&unitgroupep.CreateUnitGroupUnitEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	updateUnitEndpoint := (&unitgroupep.UpdateUnitGroupUnitEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	deleteUnitEndpoint := (&unitgroupep.DeleteUnitGroupUnitEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	listUnitEndpoint := (&unitgroupep.ListUnitGroupUnitsEndpoint{}).Materialize().WithService(inner, unitGroupSvc)
	getUnitEndpoint := (&unitgroupep.GetUnitGroupUnitEndpoint{}).Materialize().WithService(inner, unitGroupSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		getEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		listUnitEndpoint,
		getUnitEndpoint,
		createUnitEndpoint,
		updateUnitEndpoint,
		deleteUnitEndpoint,
	}

	return &UnitGroupsEndpointGroup{inner}
}
