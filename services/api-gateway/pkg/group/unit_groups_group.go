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

	listEndpoint := apiendpoint.From(&unitgroupep.ListUnitGroupsEndpoint{}).WithService(inner, unitGroupSvc)
	retrieveEndpoint := apiendpoint.From(&unitgroupep.RetrieveUnitGroupEndpoint{}).WithService(inner, unitGroupSvc)
	createEndpoint := apiendpoint.From(&unitgroupep.CreateUnitGroupEndpoint{}).WithService(inner, unitGroupSvc)
	updateEndpoint := apiendpoint.From(&unitgroupep.UpdateUnitGroupEndpoint{}).WithService(inner, unitGroupSvc)
	deleteEndpoint := apiendpoint.From(&unitgroupep.DeleteUnitGroupEndpoint{}).WithService(inner, unitGroupSvc)
	createUnitEndpoint := apiendpoint.From(&unitgroupep.CreateUnitGroupUnitEndpoint{}).WithService(inner, unitGroupSvc)
	updateUnitEndpoint := apiendpoint.From(&unitgroupep.UpdateUnitGroupUnitEndpoint{}).WithService(inner, unitGroupSvc)
	deleteUnitEndpoint := apiendpoint.From(&unitgroupep.DeleteUnitGroupUnitEndpoint{}).WithService(inner, unitGroupSvc)
	listUnitEndpoint := apiendpoint.From(&unitgroupep.ListUnitGroupUnitsEndpoint{}).WithService(inner, unitGroupSvc)
	getUnitEndpoint := apiendpoint.From(&unitgroupep.RetrieveUnitGroupUnitEndpoint{}).WithService(inner, unitGroupSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
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
