package httpgroup

import (
	"fmt"

	carrierep "github.com/augno/api/services/api-gateway/endpoints/carriers"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type CarriersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CarriersEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *CarriersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("carriers endpoint group: core client is required")
	}
	return nil
}

func (*CarriersEndpointGroup) Materialize(config *CarriersEndpointGroupConfig) *CarriersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	carrierSvc := carrierep.NewCarrierSvc(&carrierep.CarrierSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Carriers Management",
		Description:  "List and manage carriers and their Shippo integrations.",
		ResourceType: &apiresource.Carrier{},
	}

	listCarriersEndpoint := (&carrierep.ListCarriersEndpoint{}).Materialize().WithService(inner, carrierSvc)
	getCarrierEndpoint := (&carrierep.GetCarrierEndpoint{}).Materialize().WithService(inner, carrierSvc)
	createCarrierEndpoint := (&carrierep.CreateCarrierEndpoint{}).Materialize().WithService(inner, carrierSvc)
	updateCarrierEndpoint := (&carrierep.UpdateCarrierEndpoint{}).Materialize().WithService(inner, carrierSvc)
	deleteCarrierEndpoint := (&carrierep.DeleteCarrierEndpoint{}).Materialize().WithService(inner, carrierSvc)
	initiateOAuthEndpoint := (&carrierep.InitiateOAuthEndpoint{}).Materialize().WithService(inner, carrierSvc)
	getOAuthStatusEndpoint := (&carrierep.GetOAuthStatusEndpoint{}).Materialize().WithService(inner, carrierSvc)
	syncOptionsEndpoint := (&carrierep.SyncOptionsEndpoint{}).Materialize().WithService(inner, carrierSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listCarriersEndpoint,
		getCarrierEndpoint,
		createCarrierEndpoint,
		updateCarrierEndpoint,
		deleteCarrierEndpoint,
		initiateOAuthEndpoint,
		getOAuthStatusEndpoint,
		syncOptionsEndpoint,
	}

	return &CarriersEndpointGroup{inner}
}
