package httpgroup

import (
	"fmt"

	purchaseorderep "github.com/augno/api/services/api-gateway/endpoints/purchase-orders"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PurchaseOrdersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PurchaseOrdersEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PurchaseOrdersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("purchase orders endpoint group: core client is required")
	}
	return nil
}

func (*PurchaseOrdersEndpointGroup) Materialize(config *PurchaseOrdersEndpointGroupConfig) *PurchaseOrdersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := purchaseorderep.NewPurchaseOrderSvc(&purchaseorderep.PurchaseOrderSvcConfig{
		CoreClient:  config.CoreClient.Purchase,
		SalesClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Purchase Orders",
		Description:  "List, view, create, update, and delete purchase orders.",
		ResourceType: &apiresource.PurchaseOrder{},
	}

	listEndpoint := apiendpoint.From(&purchaseorderep.ListPurchaseOrdersEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&purchaseorderep.RetrievePurchaseOrderEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&purchaseorderep.CreatePurchaseOrderEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&purchaseorderep.UpdatePurchaseOrderEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&purchaseorderep.DeletePurchaseOrderEndpoint{}).WithService(inner, svc)
	bulkDeleteEndpoint := apiendpoint.From(&purchaseorderep.BulkDeletePurchaseOrdersEndpoint{}).WithService(inner, svc)
	changeStatusEndpoint := apiendpoint.From(&purchaseorderep.ChangePurchaseOrderStatusEndpoint{}).WithService(inner, svc)
	createLineEndpoint := apiendpoint.From(&purchaseorderep.CreatePurchaseOrderLineEndpoint{}).WithService(inner, svc)
	updateLineEndpoint := apiendpoint.From(&purchaseorderep.UpdatePurchaseOrderLineEndpoint{}).WithService(inner, svc)
	deleteLineEndpoint := apiendpoint.From(&purchaseorderep.DeletePurchaseOrderLineEndpoint{}).WithService(inner, svc)
	listStatusesEndpoint := apiendpoint.From(&purchaseorderep.ListPurchaseOrderStatusesEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
		changeStatusEndpoint,
		createLineEndpoint,
		updateLineEndpoint,
		deleteLineEndpoint,
		listStatusesEndpoint,
	}

	return &PurchaseOrdersEndpointGroup{inner}
}
