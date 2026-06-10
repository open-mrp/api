package httpgroup

import (
	"fmt"

	accountep "github.com/augno/api/services/api-gateway/endpoints/accounts"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("accounts endpoint group: core client is required")
	}
	return nil
}

func (*AccountsEndpointGroup) Materialize(config *AccountsEndpointGroupConfig) *AccountsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	accountSvc := accountep.NewAccountSvc(&accountep.AccountSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Management",
		Description:  "Manage account details, branding, portal, and logo.",
		ResourceType: &apiresource.Account{},
	}

	getAccountEndpoint := apiendpoint.From(&accountep.RetrieveAccountEndpoint{}).WithService(inner, accountSvc)
	getAccountBySlugEndpoint := apiendpoint.From(&accountep.RetrieveAccountBySlugEndpoint{}).WithService(inner, accountSvc)
	updateAccountEndpoint := apiendpoint.From(&accountep.UpdateAccountEndpoint{}).WithService(inner, accountSvc)
	uploadAccountPhotoEndpoint := apiendpoint.From(&accountep.UploadAccountPhotoEndpoint{}).WithService(inner, accountSvc)
	getAccountLogoURLEndpoint := apiendpoint.From(&accountep.GetAccountLogoURLEndpoint{}).WithService(inner, accountSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getAccountEndpoint,
		getAccountBySlugEndpoint,
		updateAccountEndpoint,
		uploadAccountPhotoEndpoint,
		getAccountLogoURLEndpoint,
	}

	return &AccountsEndpointGroup{inner}
}
