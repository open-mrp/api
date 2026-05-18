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

	listCarriersEndpoint := apiendpoint.From(&carrierep.ListCarriersEndpoint{}).WithService(inner, carrierSvc)
	getCarrierEndpoint := apiendpoint.From(&carrierep.RetrieveCarrierEndpoint{}).WithService(inner, carrierSvc)
	createCarrierEndpoint := apiendpoint.From(&carrierep.CreateCarrierEndpoint{}).WithService(inner, carrierSvc)
	updateCarrierEndpoint := apiendpoint.From(&carrierep.UpdateCarrierEndpoint{}).WithService(inner, carrierSvc)
	deleteCarrierEndpoint := apiendpoint.From(&carrierep.DeleteCarrierEndpoint{}).WithService(inner, carrierSvc)
	initiateOAuthEndpoint := apiendpoint.From(&carrierep.InitiateOAuthEndpoint{}).WithService(inner, carrierSvc)
	getOAuthStatusEndpoint := apiendpoint.From(&carrierep.GetOAuthStatusEndpoint{}).WithService(inner, carrierSvc)
	syncOptionsEndpoint := apiendpoint.From(&carrierep.SyncOptionsEndpoint{}).WithService(inner, carrierSvc)

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
