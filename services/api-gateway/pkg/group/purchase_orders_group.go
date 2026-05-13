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
		ResourceType: &apiresource.PurchaseOrderDetail{},
	}

	listEndpoint := (&purchaseorderep.ListPurchaseOrdersEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&purchaseorderep.RetrievePurchaseOrderEndpoint{}).Materialize().WithService(inner, svc)
	createEndpoint := (&purchaseorderep.CreatePurchaseOrderEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&purchaseorderep.UpdatePurchaseOrderEndpoint{}).Materialize().WithService(inner, svc)
	deleteEndpoint := (&purchaseorderep.DeletePurchaseOrderEndpoint{}).Materialize().WithService(inner, svc)
	bulkDeleteEndpoint := (&purchaseorderep.BulkDeletePurchaseOrdersEndpoint{}).Materialize().WithService(inner, svc)
	changeStatusEndpoint := (&purchaseorderep.ChangePurchaseOrderStatusEndpoint{}).Materialize().WithService(inner, svc)
	createLineEndpoint := (&purchaseorderep.CreatePurchaseOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	updateLineEndpoint := (&purchaseorderep.UpdatePurchaseOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	deleteLineEndpoint := (&purchaseorderep.DeletePurchaseOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	listStatusesEndpoint := (&purchaseorderep.ListPurchaseOrderStatusesEndpoint{}).Materialize().WithService(inner, svc)

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
