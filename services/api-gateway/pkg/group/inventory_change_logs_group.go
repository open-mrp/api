package httpgroup

import (
	"fmt"

	inventorychangelogep "github.com/open-mrp/api/services/api-gateway/endpoints/inventory-change-logs"
	inventorymovementep "github.com/open-mrp/api/services/api-gateway/endpoints/inventory-movements"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
	startExportEndpoint := apiendpoint.From(&inventorychangelogep.StartInventoryChangeLogsExportEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		exportEndpoint,
		startExportEndpoint,
	}

	return &InventoryChangeLogsEndpointGroup{inner}
}

// InventoryMovementsEndpointGroup serves inventory movements: listing, recording, bulk recording and exporting them.
type InventoryMovementsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

// InventoryLevelsEndpointGroup serves items' stock positions.
type InventoryLevelsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

// LotSizeDefaultsEndpointGroup serves the quantity each item is made in.
type LotSizeDefaultsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

func newInventoryMovementSvc(config *InventoryChangeLogsEndpointGroupConfig) inventorymovementep.InventoryMovementSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return inventorymovementep.NewInventoryMovementSvc(&inventorymovementep.InventoryMovementSvcConfig{
		CoreClient:      config.CoreClient.Client,
		InventoryClient: config.CoreClient.Inventory,
	})
}

func (*InventoryMovementsEndpointGroup) Materialize(config *InventoryChangeLogsEndpointGroupConfig) *InventoryMovementsEndpointGroup {
	svc := newInventoryMovementSvc(config)
	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Inventory Movements",
		Description:  "Every change to an item's stock: receipts, shipments, production, transfers, adjustments and counts.",
		ResourceType: &apiresource.InventoryMovement{},
	}
	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&inventorymovementep.ListInventoryMovementsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&inventorymovementep.CreateInventoryMovementEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&inventorymovementep.RetrieveInventoryMovementEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&inventorymovementep.BulkCreateInventoryMovementsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&inventorymovementep.ExportInventoryMovementsEndpoint{}).WithService(inner, svc),
	}
	return &InventoryMovementsEndpointGroup{inner}
}

func (*InventoryLevelsEndpointGroup) Materialize(config *InventoryChangeLogsEndpointGroupConfig) *InventoryLevelsEndpointGroup {
	svc := newInventoryMovementSvc(config)
	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Inventory Levels",
		Description:  "Items' stock positions: on hand, reserved, available to promise and short.",
		ResourceType: &apiresource.InventoryLevel{},
	}
	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&inventorymovementep.ListInventoryLevelsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&inventorymovementep.RetrieveItemInventoryLevelEndpoint{}).WithService(inner, svc),
	}
	return &InventoryLevelsEndpointGroup{inner}
}

func (*LotSizeDefaultsEndpointGroup) Materialize(config *InventoryChangeLogsEndpointGroupConfig) *LotSizeDefaultsEndpointGroup {
	svc := newInventoryMovementSvc(config)
	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Lot Size Defaults",
		Description:  "The quantity each item is made in, resolved from the item, its category and the account.",
		ResourceType: &apiresource.LotSizeDefault{},
	}
	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&inventorymovementep.RetrieveLotSizeDefaultEndpoint{}).WithService(inner, svc),
	}
	return &LotSizeDefaultsEndpointGroup{inner}
}
