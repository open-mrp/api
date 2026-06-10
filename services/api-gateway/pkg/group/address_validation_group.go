package httpgroup

import (
	"fmt"

	addressvalidationep "github.com/augno/api/services/api-gateway/endpoints/address-validation"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AddressValidationEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AddressValidationEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AddressValidationEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("address validation endpoint group: core client is required")
	}
	return nil
}

func (*AddressValidationEndpointGroup) Materialize(config *AddressValidationEndpointGroupConfig) *AddressValidationEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	addressValidationSvc := addressvalidationep.NewAddressValidationSvc(&addressvalidationep.AddressValidationSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Address Validation",
		Description:  "Autocomplete, look up details, and validate addresses.",
		ResourceType: &apiresource.ValidatedAddress{},
	}

	autocompleteEndpoint := apiendpoint.From(&addressvalidationep.AutocompleteAddressEndpoint{}).WithService(inner, addressValidationSvc)
	detailsEndpoint := apiendpoint.From(&addressvalidationep.RetrieveAddressDetailsEndpoint{}).WithService(inner, addressValidationSvc)
	validateEndpoint := apiendpoint.From(&addressvalidationep.ValidateAddressEndpoint{}).WithService(inner, addressValidationSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		autocompleteEndpoint,
		detailsEndpoint,
		validateEndpoint,
	}

	return &AddressValidationEndpointGroup{inner}
}
