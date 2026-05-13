package httpgroup

import (
	"fmt"

	volumediscountep "github.com/augno/api/services/api-gateway/endpoints/volume-discounts"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type VolumeDiscountsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type VolumeDiscountsEndpointGroupConfig struct {
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

	listEndpoint := (&volumediscountep.ListVolumeDiscountsEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&volumediscountep.RetrieveVolumeDiscountEndpoint{}).Materialize().WithService(inner, svc)
	createEndpoint := (&volumediscountep.CreateVolumeDiscountEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&volumediscountep.UpdateVolumeDiscountEndpoint{}).Materialize().WithService(inner, svc)
	deleteEndpoint := (&volumediscountep.DeleteVolumeDiscountEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &VolumeDiscountsEndpointGroup{inner}
}
