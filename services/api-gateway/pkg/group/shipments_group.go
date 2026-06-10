package httpgroup

import (
	"fmt"

	shipmentep "github.com/augno/api/services/api-gateway/endpoints/shipments"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ShipmentsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ShipmentsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ShipmentsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipments endpoint group: core client is required")
	}
	return nil
}

func (*ShipmentsEndpointGroup) Materialize(config *ShipmentsEndpointGroupConfig) *ShipmentsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	shipmentSvc := shipmentep.NewShipmentSvc(&shipmentep.ShipmentSvcConfig{
		CoreClient: config.CoreClient.Shipping,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Shipment Management",
		Description:  "List and manage shipments, shipment lines, and shipping operations.",
		ResourceType: &apiresource.Shipment{},
	}

	listShipmentsEndpoint := apiendpoint.From(&shipmentep.ListShipmentsEndpoint{}).WithService(inner, shipmentSvc)
	getShipmentEndpoint := apiendpoint.From(&shipmentep.RetrieveShipmentEndpoint{}).WithService(inner, shipmentSvc)
	updateShipmentEndpoint := apiendpoint.From(&shipmentep.UpdateShipmentEndpoint{}).WithService(inner, shipmentSvc)
	deleteShipmentEndpoint := apiendpoint.From(&shipmentep.DeleteShipmentEndpoint{}).WithService(inner, shipmentSvc)
	shipShipmentEndpoint := apiendpoint.From(&shipmentep.ShipShipmentEndpoint{}).WithService(inner, shipmentSvc)
	voidShipmentEndpoint := apiendpoint.From(&shipmentep.VoidShipmentEndpoint{}).WithService(inner, shipmentSvc)
	estimateRateEndpoint := apiendpoint.From(&shipmentep.EstimateRateEndpoint{}).WithService(inner, shipmentSvc)
	rateShopEndpoint := apiendpoint.From(&shipmentep.RateShopEndpoint{}).WithService(inner, shipmentSvc)
	listShipmentLinesEndpoint := apiendpoint.From(&shipmentep.ListShipmentLinesEndpoint{}).WithService(inner, shipmentSvc)
	getShipmentLineEndpoint := apiendpoint.From(&shipmentep.RetrieveShipmentLineEndpoint{}).WithService(inner, shipmentSvc)
	createShipmentLineEndpoint := apiendpoint.From(&shipmentep.CreateShipmentLineEndpoint{}).WithService(inner, shipmentSvc)
	updateShipmentLineEndpoint := apiendpoint.From(&shipmentep.UpdateShipmentLineEndpoint{}).WithService(inner, shipmentSvc)
	deleteShipmentLineEndpoint := apiendpoint.From(&shipmentep.DeleteShipmentLineEndpoint{}).WithService(inner, shipmentSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listShipmentsEndpoint,
		getShipmentEndpoint,
		updateShipmentEndpoint,
		deleteShipmentEndpoint,
		shipShipmentEndpoint,
		voidShipmentEndpoint,
		estimateRateEndpoint,
		rateShopEndpoint,
		listShipmentLinesEndpoint,
		getShipmentLineEndpoint,
		createShipmentLineEndpoint,
		updateShipmentLineEndpoint,
		deleteShipmentLineEndpoint,
	}

	return &ShipmentsEndpointGroup{inner}
}
