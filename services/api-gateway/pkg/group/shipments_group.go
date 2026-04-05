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
		ResourceType: &apiresource.ShipmentDetail{},
	}

	listShipmentsEndpoint := (&shipmentep.ListShipmentsEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	getShipmentEndpoint := (&shipmentep.GetShipmentEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	updateShipmentEndpoint := (&shipmentep.UpdateShipmentEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	deleteShipmentEndpoint := (&shipmentep.DeleteShipmentEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	shipShipmentEndpoint := (&shipmentep.ShipShipmentEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	voidShipmentEndpoint := (&shipmentep.VoidShipmentEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	estimateRateEndpoint := (&shipmentep.EstimateRateEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	rateShopEndpoint := (&shipmentep.RateShopEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	listShipmentLinesEndpoint := (&shipmentep.ListShipmentLinesEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	getShipmentLineEndpoint := (&shipmentep.GetShipmentLineEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	createShipmentLineEndpoint := (&shipmentep.CreateShipmentLineEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	updateShipmentLineEndpoint := (&shipmentep.UpdateShipmentLineEndpoint{}).Materialize().WithService(inner, shipmentSvc)
	deleteShipmentLineEndpoint := (&shipmentep.DeleteShipmentLineEndpoint{}).Materialize().WithService(inner, shipmentSvc)

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
