package httpgroup

import (
	"fmt"

	addressep "github.com/open-mrp/api/services/api-gateway/endpoints/addresses"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type AddressesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AddressesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Address",
		Description:  "List and manage addresses for accounts.",
		ResourceType: &apiresource.Address{},
	}

	listAddressesEndpoint := apiendpoint.From(&addressep.ListAddressesEndpoint{}).WithService(inner, addressSvc)
	getAddressEndpoint := apiendpoint.From(&addressep.RetrieveAddressEndpoint{}).WithService(inner, addressSvc)
	createAddressEndpoint := apiendpoint.From(&addressep.CreateAddressEndpoint{}).WithService(inner, addressSvc)
	updateAddressEndpoint := apiendpoint.From(&addressep.UpdateAddressEndpoint{}).WithService(inner, addressSvc)
	deleteAddressEndpoint := apiendpoint.From(&addressep.DeleteAddressEndpoint{}).WithService(inner, addressSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listAddressesEndpoint,
		getAddressEndpoint,
		createAddressEndpoint,
		updateAddressEndpoint,
		deleteAddressEndpoint,
	}

	return &AddressesEndpointGroup{inner}
}
