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

	listEndpoint := (&accountgroupproductlineaccessep.ListAccountGroupProductLineAccessEndpoint{}).Materialize().WithService(inner, svc)
	getEndpoint := (&accountgroupproductlineaccessep.GetAccountGroupProductLineAccessEndpoint{}).Materialize().WithService(inner, svc)
	createEndpoint := (&accountgroupproductlineaccessep.CreateAccountGroupProductLineAccessEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&accountgroupproductlineaccessep.UpdateAccountGroupProductLineAccessEndpoint{}).Materialize().WithService(inner, svc)
	deleteEndpoint := (&accountgroupproductlineaccessep.DeleteAccountGroupProductLineAccessEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		getEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &AccountGroupProductLineAccessEndpointGroup{inner}
}
