package httpgroup

import (
	"fmt"

	tenancyep "github.com/augno/api/services/api-gateway/endpoints/tenancy"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type TenancyEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type TenancyEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *TenancyEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("tenancy endpoint group: core client is required")
	}
	return nil
}

func (*TenancyEndpointGroup) Materialize(config *TenancyEndpointGroupConfig) *TenancyEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := tenancyep.NewTenancySvc(&tenancyep.TenancySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Tenancy",
		Description:  "Manage the authenticated user's tenancy context, including account switching and customer account access.",
		ResourceType: &apiresource.Tenancy{},
	}

	getTenancyEndpoint := apiendpoint.From(&tenancyep.GetTenancyEndpoint{}).WithService(inner, svc)
	switchAccountEndpoint := apiendpoint.From(&tenancyep.SwitchAccountEndpoint{}).WithService(inner, svc)
	getCurrentUserEndpoint := apiendpoint.From(&tenancyep.GetCurrentUserEndpoint{}).WithService(inner, svc)
	listCustomerAccountsEndpoint := apiendpoint.From(&tenancyep.ListCustomerAccountsEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getTenancyEndpoint,
		switchAccountEndpoint,
		getCurrentUserEndpoint,
		listCustomerAccountsEndpoint,
	}

	return &TenancyEndpointGroup{inner}
}
