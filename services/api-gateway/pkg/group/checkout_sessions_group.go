package httpgroup

import (
	"fmt"

	checkoutsessionep "github.com/open-mrp/api/services/api-gateway/endpoints/checkout-sessions"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
)

type CheckoutSessionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CheckoutSessionsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *CheckoutSessionsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("checkout sessions endpoint group: core client is required")
	}
	return nil
}

func (*CheckoutSessionsEndpointGroup) Materialize(config *CheckoutSessionsEndpointGroupConfig) *CheckoutSessionsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := checkoutsessionep.NewCheckoutSessionSvc(&checkoutsessionep.CheckoutSessionSvcConfig{
		CoreSalesClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Checkout Sessions",
		Description:  "Create customer checkout sessions.",
		ResourceType: &checkoutsessionep.CheckoutSessionResponse{},
	}

	createEndpoint := apiendpoint.From(&checkoutsessionep.CreateCheckoutSessionEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		createEndpoint,
	}

	return &CheckoutSessionsEndpointGroup{inner}
}
