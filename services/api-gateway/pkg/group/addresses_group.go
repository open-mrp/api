package httpgroup

import (
	"fmt"

	addressep "github.com/augno/api/services/api-gateway/endpoints/addresses"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AddressesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AddressesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AddressesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("addresses endpoint group: core client is required")
	}
	return nil
}

func (*AddressesEndpointGroup) Materialize(config *AddressesEndpointGroupConfig) *AddressesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	addressSvc := addressep.NewAddressSvc(&addressep.AddressSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Address Management",
		Description:  "List and manage addresses for accounts.",
		ResourceType: &apiresource.Address{},
	}

	listAddressesEndpoint := (&addressep.ListAddressesEndpoint{}).Materialize().WithService(inner, addressSvc)
	getAddressEndpoint := (&addressep.GetAddressEndpoint{}).Materialize().WithService(inner, addressSvc)
	createAddressEndpoint := (&addressep.CreateAddressEndpoint{}).Materialize().WithService(inner, addressSvc)
	updateAddressEndpoint := (&addressep.UpdateAddressEndpoint{}).Materialize().WithService(inner, addressSvc)
	deleteAddressEndpoint := (&addressep.DeleteAddressEndpoint{}).Materialize().WithService(inner, addressSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listAddressesEndpoint,
		getAddressEndpoint,
		createAddressEndpoint,
		updateAddressEndpoint,
		deleteAddressEndpoint,
	}

	return &AddressesEndpointGroup{inner}
}
