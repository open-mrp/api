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
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Territory",
		Description:  "List and manage territories for accounts.",
		ResourceType: &apiresource.Territory{},
	}

	listTerritoriesEndpoint := apiendpoint.From(&territoryep.ListTerritoriesEndpoint{}).WithService(inner, territorySvc)
	getTerritoryEndpoint := apiendpoint.From(&territoryep.RetrieveTerritoryEndpoint{}).WithService(inner, territorySvc)
	createTerritoryEndpoint := apiendpoint.From(&territoryep.CreateTerritoryEndpoint{}).WithService(inner, territorySvc)
	updateTerritoryEndpoint := apiendpoint.From(&territoryep.UpdateTerritoryEndpoint{}).WithService(inner, territorySvc)
	deleteTerritoryEndpoint := apiendpoint.From(&territoryep.DeleteTerritoryEndpoint{}).WithService(inner, territorySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listTerritoriesEndpoint,
		getTerritoryEndpoint,
		createTerritoryEndpoint,
		updateTerritoryEndpoint,
		deleteTerritoryEndpoint,
	}

	return &TerritoriesEndpointGroup{inner}
}
