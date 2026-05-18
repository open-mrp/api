package httpgroup

import (
	"fmt"

	shippingcaseep "github.com/augno/api/services/api-gateway/endpoints/shipping-cases"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ShippingCasesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ShippingCasesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ShippingCasesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipping cases endpoint group: core client is required")
	}
	return nil
}

func (*ShippingCasesEndpointGroup) Materialize(config *ShippingCasesEndpointGroupConfig) *ShippingCasesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	shippingCaseSvc := shippingcaseep.NewShippingCaseSvc(&shippingcaseep.ShippingCaseSvcConfig{
		CoreClient: config.CoreClient.ShippingCase,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Shipping Cases",
		Description:  "Manage shipping cases within shipments.",
		ResourceType: &apiresource.ShippingCase{},
	}

	getShippingCaseEndpoint := apiendpoint.From(&shippingcaseep.RetrieveShippingCaseEndpoint{}).WithService(inner, shippingCaseSvc)
	updateShippingCaseEndpoint := apiendpoint.From(&shippingcaseep.UpdateShippingCaseEndpoint{}).WithService(inner, shippingCaseSvc)
	deleteShippingCaseEndpoint := apiendpoint.From(&shippingcaseep.DeleteShippingCaseEndpoint{}).WithService(inner, shippingCaseSvc)
	getShippingCaseLabelEndpoint := apiendpoint.From(&shippingcaseep.GetShippingCaseLabelEndpoint{}).WithService(inner, shippingCaseSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getShippingCaseEndpoint,
		updateShippingCaseEndpoint,
		deleteShippingCaseEndpoint,
		getShippingCaseLabelEndpoint,
	}

	return &ShippingCasesEndpointGroup{inner}
}
