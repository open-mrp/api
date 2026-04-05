package httpgroup

import (
	"fmt"

	inventorychangelogep "github.com/augno/api/services/api-gateway/endpoints/inventory-change-logs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type InventoryChangeLogsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type InventoryChangeLogsEndpointGroupConfig struct {
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

	listEndpoint := (&inventorychangelogep.ListInventoryChangeLogsEndpoint{}).Materialize().WithService(inner, svc)
	getEndpoint := (&inventorychangelogep.GetInventoryChangeLogEndpoint{}).Materialize().WithService(inner, svc)
	exportEndpoint := (&inventorychangelogep.ExportInventoryChangeLogsEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		getEndpoint,
		exportEndpoint,
	}

	return &InventoryChangeLogsEndpointGroup{inner}
}
