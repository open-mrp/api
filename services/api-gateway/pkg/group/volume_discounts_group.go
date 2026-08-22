package httpgroup

import (
	"fmt"

	volumediscountep "github.com/open-mrp/api/services/api-gateway/endpoints/volume-discounts"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type VolumeDiscountsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type VolumeDiscountsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *VolumeDiscountsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("volume discounts endpoint group: core client is required")
	}
	return nil
}

func (*VolumeDiscountsEndpointGroup) Materialize(config *VolumeDiscountsEndpointGroupConfig) *VolumeDiscountsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := volumediscountep.NewVolumeDiscountSvc(&volumediscountep.VolumeDiscountSvcConfig{
		CoreClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Volume Discounts",
		Description:  "List and manage volume discounts.",
		ResourceType: &apiresource.VolumeDiscount{},
	}

	listEndpoint := apiendpoint.From(&volumediscountep.ListVolumeDiscountsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&volumediscountep.RetrieveVolumeDiscountEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&volumediscountep.CreateVolumeDiscountEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&volumediscountep.UpdateVolumeDiscountEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&volumediscountep.DeleteVolumeDiscountEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &VolumeDiscountsEndpointGroup{inner}
}
