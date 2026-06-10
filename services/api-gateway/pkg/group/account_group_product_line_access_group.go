package httpgroup

import (
	"fmt"

	accountgroupproductlineaccessep "github.com/augno/api/services/api-gateway/endpoints/account-group-product-line-access"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountGroupProductLineAccessEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountGroupProductLineAccessEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountGroupProductLineAccessEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account group product line access endpoint group: core client is required")
	}
	return nil
}

func (*AccountGroupProductLineAccessEndpointGroup) Materialize(config *AccountGroupProductLineAccessEndpointGroupConfig) *AccountGroupProductLineAccessEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := accountgroupproductlineaccessep.NewAccountGroupProductLineAccessSvc(&accountgroupproductlineaccessep.AccountGroupProductLineAccessSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Group Product Line Access",
		Description:  "Manage product line access for account groups.",
		ResourceType: &apiresource.AccountGroupProductLineAccess{},
	}

	listEndpoint := apiendpoint.From(&accountgroupproductlineaccessep.ListAccountGroupProductLineAccessEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&accountgroupproductlineaccessep.RetrieveAccountGroupProductLineAccessEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&accountgroupproductlineaccessep.CreateAccountGroupProductLineAccessEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&accountgroupproductlineaccessep.UpdateAccountGroupProductLineAccessEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&accountgroupproductlineaccessep.DeleteAccountGroupProductLineAccessEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &AccountGroupProductLineAccessEndpointGroup{inner}
}
