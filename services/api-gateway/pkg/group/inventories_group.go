package httpgroup

import (
	"fmt"

	inventoryep "github.com/open-mrp/api/services/api-gateway/endpoints/inventories"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type InventoriesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type InventoriesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *InventoriesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("inventories endpoint group: core client is required")
	}
	return nil
}

func (*InventoriesEndpointGroup) Materialize(config *InventoriesEndpointGroupConfig) *InventoriesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	inventorySvc := inventoryep.NewInventorySvc(&inventoryep.InventorySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Inventories",
		Description:  "List item inventories with on-hand quantities.",
		ResourceType: &apiresource.InventoryItem{},
	}

	listInventoriesEndpoint := apiendpoint.From(&inventoryep.ListInventoriesEndpoint{}).WithService(inner, inventorySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listInventoriesEndpoint,
	}

	return &InventoriesEndpointGroup{inner}
}
