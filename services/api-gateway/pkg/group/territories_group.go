package httpgroup

import (
	"fmt"

	territoryep "github.com/augno/api/services/api-gateway/endpoints/territories"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type TerritoriesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type TerritoriesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *TerritoriesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("territories endpoint group: core client is required")
	}
	return nil
}

func (*TerritoriesEndpointGroup) Materialize(config *TerritoriesEndpointGroupConfig) *TerritoriesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	territorySvc := territoryep.NewTerritorySvc(&territoryep.TerritorySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Territory Management",
		Description:  "List and manage territories for accounts.",
		ResourceType: &apiresource.Territory{},
	}

	listTerritoriesEndpoint := (&territoryep.ListTerritoriesEndpoint{}).Materialize().WithService(inner, territorySvc)
	getTerritoryEndpoint := (&territoryep.GetTerritoryEndpoint{}).Materialize().WithService(inner, territorySvc)
	createTerritoryEndpoint := (&territoryep.CreateTerritoryEndpoint{}).Materialize().WithService(inner, territorySvc)
	updateTerritoryEndpoint := (&territoryep.UpdateTerritoryEndpoint{}).Materialize().WithService(inner, territorySvc)
	deleteTerritoryEndpoint := (&territoryep.DeleteTerritoryEndpoint{}).Materialize().WithService(inner, territorySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listTerritoriesEndpoint,
		getTerritoryEndpoint,
		createTerritoryEndpoint,
		updateTerritoryEndpoint,
		deleteTerritoryEndpoint,
	}

	return &TerritoriesEndpointGroup{inner}
}
