package httpgroup

import (
	"fmt"

	inventorychangelogep "github.com/open-mrp/api/services/api-gateway/endpoints/inventory-change-logs"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type InventoryChangeLogsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type InventoryChangeLogsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *InventoryChangeLogsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("inventory change logs endpoint group: core client is required")
	}
	return nil
}

func (*InventoryChangeLogsEndpointGroup) Materialize(config *InventoryChangeLogsEndpointGroupConfig) *InventoryChangeLogsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := inventorychangelogep.NewInventoryChangeLogSvc(&inventorychangelogep.InventoryChangeLogSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Inventory Change Logs",
		Description:  "List and export inventory change logs.",
		ResourceType: &apiresource.InventoryChangeLog{},
	}

	listEndpoint := apiendpoint.From(&inventorychangelogep.ListInventoryChangeLogsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&inventorychangelogep.RetrieveInventoryChangeLogEndpoint{}).WithService(inner, svc)
	exportEndpoint := apiendpoint.From(&inventorychangelogep.ExportInventoryChangeLogsEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		exportEndpoint,
	}

	return &InventoryChangeLogsEndpointGroup{inner}
}
