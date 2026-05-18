package httpgroup

import (
	"fmt"

	childaccountep "github.com/augno/api/services/api-gateway/endpoints/child-accounts"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ChildAccountsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ChildAccountsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ChildAccountsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("child accounts endpoint group: core client is required")
	}
	return nil
}

func (*ChildAccountsEndpointGroup) Materialize(config *ChildAccountsEndpointGroupConfig) *ChildAccountsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	childAccountSvc := childaccountep.NewChildAccountSvc(&childaccountep.ChildAccountSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Child Accounts Management",
		Description:  "Manage parent-child relationships between customer accounts.",
		ResourceType: &apiresource.ChildAccount{},
	}

	listEndpoint := apiendpoint.From(&childaccountep.ListChildAccountsEndpoint{}).WithService(inner, childAccountSvc)
	addEndpoint := apiendpoint.From(&childaccountep.AddChildAccountEndpoint{}).WithService(inner, childAccountSvc)
	removeEndpoint := apiendpoint.From(&childaccountep.RemoveChildAccountEndpoint{}).WithService(inner, childAccountSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		addEndpoint,
		removeEndpoint,
	}

	return &ChildAccountsEndpointGroup{inner}
}
