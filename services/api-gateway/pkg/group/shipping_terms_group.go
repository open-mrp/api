package httpgroup

import (
	"fmt"

	shippingtermep "github.com/augno/api/services/api-gateway/endpoints/shipping-terms"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ShippingTermsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ShippingTermsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ShippingTermsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipping terms endpoint group: core client is required")
	}
	return nil
}

func (*ShippingTermsEndpointGroup) Materialize(config *ShippingTermsEndpointGroupConfig) *ShippingTermsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	shippingTermSvc := shippingtermep.NewShippingTermSvc(&shippingtermep.ShippingTermSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Shipping Terms Management",
		Description:  "List and manage shipping terms.",
		ResourceType: &apiresource.ShippingTerm{},
	}

	listShippingTermsEndpoint := apiendpoint.From(&shippingtermep.ListShippingTermsEndpoint{}).WithService(inner, shippingTermSvc)
	getShippingTermEndpoint := apiendpoint.From(&shippingtermep.RetrieveShippingTermEndpoint{}).WithService(inner, shippingTermSvc)
	createShippingTermEndpoint := apiendpoint.From(&shippingtermep.CreateShippingTermEndpoint{}).WithService(inner, shippingTermSvc)
	updateShippingTermEndpoint := apiendpoint.From(&shippingtermep.UpdateShippingTermEndpoint{}).WithService(inner, shippingTermSvc)
	deleteShippingTermEndpoint := apiendpoint.From(&shippingtermep.DeleteShippingTermEndpoint{}).WithService(inner, shippingTermSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listShippingTermsEndpoint,
		getShippingTermEndpoint,
		createShippingTermEndpoint,
		updateShippingTermEndpoint,
		deleteShippingTermEndpoint,
	}

	return &ShippingTermsEndpointGroup{inner}
}
