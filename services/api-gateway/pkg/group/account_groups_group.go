package httpgroup

import (
	"fmt"

	accountgroupep "github.com/augno/api/services/api-gateway/endpoints/account-groups"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountGroupsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountGroupsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountGroupsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account groups endpoint group: core client is required")
	}
	return nil
}

func (*AccountGroupsEndpointGroup) Materialize(config *AccountGroupsEndpointGroupConfig) *AccountGroupsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	accountGroupSvc := accountgroupep.NewAccountGroupSvc(&accountgroupep.AccountGroupSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Groups",
		Description:  "List and manage account groups.",
		ResourceType: &apiresource.AccountGroup{},
	}

	listEndpoint := (&accountgroupep.ListAccountGroupsEndpoint{}).Materialize().WithService(inner, accountGroupSvc)
	retrieveEndpoint := (&accountgroupep.RetrieveAccountGroupEndpoint{}).Materialize().WithService(inner, accountGroupSvc)
	createEndpoint := (&accountgroupep.CreateAccountGroupEndpoint{}).Materialize().WithService(inner, accountGroupSvc)
	updateEndpoint := (&accountgroupep.UpdateAccountGroupEndpoint{}).Materialize().WithService(inner, accountGroupSvc)
	deleteEndpoint := (&accountgroupep.DeleteAccountGroupEndpoint{}).Materialize().WithService(inner, accountGroupSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &AccountGroupsEndpointGroup{inner}
}
