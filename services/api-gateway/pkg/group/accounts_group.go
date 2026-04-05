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

	getAccountEndpoint := (&accountep.GetAccountEndpoint{}).Materialize().WithService(inner, accountSvc)
	getAccountBySlugEndpoint := (&accountep.GetAccountBySlugEndpoint{}).Materialize().WithService(inner, accountSvc)
	updateAccountEndpoint := (&accountep.UpdateAccountEndpoint{}).Materialize().WithService(inner, accountSvc)
	uploadAccountPhotoEndpoint := (&accountep.UploadAccountPhotoEndpoint{}).Materialize().WithService(inner, accountSvc)
	getAccountLogoURLEndpoint := (&accountep.GetAccountLogoURLEndpoint{}).Materialize().WithService(inner, accountSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getAccountEndpoint,
		getAccountBySlugEndpoint,
		updateAccountEndpoint,
		uploadAccountPhotoEndpoint,
		getAccountLogoURLEndpoint,
	}

	return &AccountsEndpointGroup{inner}
}
