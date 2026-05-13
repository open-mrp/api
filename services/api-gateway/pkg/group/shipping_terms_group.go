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

	listShippingTermsEndpoint := (&shippingtermep.ListShippingTermsEndpoint{}).Materialize().WithService(inner, shippingTermSvc)
	getShippingTermEndpoint := (&shippingtermep.RetrieveShippingTermEndpoint{}).Materialize().WithService(inner, shippingTermSvc)
	createShippingTermEndpoint := (&shippingtermep.CreateShippingTermEndpoint{}).Materialize().WithService(inner, shippingTermSvc)
	updateShippingTermEndpoint := (&shippingtermep.UpdateShippingTermEndpoint{}).Materialize().WithService(inner, shippingTermSvc)
	deleteShippingTermEndpoint := (&shippingtermep.DeleteShippingTermEndpoint{}).Materialize().WithService(inner, shippingTermSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listShippingTermsEndpoint,
		getShippingTermEndpoint,
		createShippingTermEndpoint,
		updateShippingTermEndpoint,
		deleteShippingTermEndpoint,
	}

	return &ShippingTermsEndpointGroup{inner}
}
