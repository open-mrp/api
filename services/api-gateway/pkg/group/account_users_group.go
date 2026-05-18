package httpgroup

import (
	"fmt"

	accountuserep "github.com/augno/api/services/api-gateway/endpoints/account-users"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountUsersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountUsersEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountUsersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account users endpoint group: core client is required")
	}
	return nil
}

func (*AccountUsersEndpointGroup) Materialize(config *AccountUsersEndpointGroupConfig) *AccountUsersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := accountuserep.NewAccountUserSvc(&accountuserep.AccountUserSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Users Management",
		Description:  "List and manage account users.",
		ResourceType: &apiresource.AccountUser{},
	}

	listEndpoint := apiendpoint.From(&accountuserep.ListAccountUsersEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&accountuserep.RetrieveAccountUserEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&accountuserep.CreateAccountUserEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&accountuserep.UpdateAccountUserEndpoint{}).WithService(inner, svc)
	activateEndpoint := apiendpoint.From(&accountuserep.ActivateAccountUserEndpoint{}).WithService(inner, svc)
	disableEndpoint := apiendpoint.From(&accountuserep.DisableAccountUserEndpoint{}).WithService(inner, svc)
	removeEndpoint := apiendpoint.From(&accountuserep.RemoveAccountUserEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		activateEndpoint,
		disableEndpoint,
		removeEndpoint,
	}

	return &AccountUsersEndpointGroup{inner}
}
