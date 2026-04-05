package httpgroup

import (
	"fmt"

	accountpriceep "github.com/augno/api/services/api-gateway/endpoints/account-prices"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountPricesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountPricesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountPricesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account prices endpoint group: core client is required")
	}
	return nil
}

func (*AccountPricesEndpointGroup) Materialize(config *AccountPricesEndpointGroupConfig) *AccountPricesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	accountPriceSvc := accountpriceep.NewAccountPriceSvc(&accountpriceep.AccountPriceSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Prices Management",
		Description:  "List and manage account prices.",
		ResourceType: &apiresource.AccountPrice{},
	}

	listAccountPricesEndpoint := (&accountpriceep.ListAccountPricesEndpoint{}).Materialize().WithService(inner, accountPriceSvc)
	getAccountPriceEndpoint := (&accountpriceep.GetAccountPriceEndpoint{}).Materialize().WithService(inner, accountPriceSvc)
	createAccountPriceEndpoint := (&accountpriceep.CreateAccountPriceEndpoint{}).Materialize().WithService(inner, accountPriceSvc)
	updateAccountPriceEndpoint := (&accountpriceep.UpdateAccountPriceEndpoint{}).Materialize().WithService(inner, accountPriceSvc)
	deleteAccountPriceEndpoint := (&accountpriceep.DeleteAccountPriceEndpoint{}).Materialize().WithService(inner, accountPriceSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listAccountPricesEndpoint,
		getAccountPriceEndpoint,
		createAccountPriceEndpoint,
		updateAccountPriceEndpoint,
		deleteAccountPriceEndpoint,
	}

	return &AccountPricesEndpointGroup{inner}
}
