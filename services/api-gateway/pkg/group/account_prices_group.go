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
	// CoreClient (required) is the core-service gRPC client.
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
		CoreClient:  config.CoreClient.Client,
		SalesClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Prices",
		Description:  "List and manage account prices.",
		ResourceType: &apiresource.AccountPrice{},
	}

	listAccountPricesEndpoint := apiendpoint.From(&accountpriceep.ListAccountPricesEndpoint{}).WithService(inner, accountPriceSvc)
	getAccountPriceEndpoint := apiendpoint.From(&accountpriceep.RetrieveAccountPriceEndpoint{}).WithService(inner, accountPriceSvc)
	createAccountPriceEndpoint := apiendpoint.From(&accountpriceep.CreateAccountPriceEndpoint{}).WithService(inner, accountPriceSvc)
	updateAccountPriceEndpoint := apiendpoint.From(&accountpriceep.UpdateAccountPriceEndpoint{}).WithService(inner, accountPriceSvc)
	deleteAccountPriceEndpoint := apiendpoint.From(&accountpriceep.DeleteAccountPriceEndpoint{}).WithService(inner, accountPriceSvc)
	exportPriceListEndpoint := apiendpoint.From(&accountpriceep.ExportPriceListEndpoint{}).WithService(inner, accountPriceSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listAccountPricesEndpoint,
		getAccountPriceEndpoint,
		createAccountPriceEndpoint,
		updateAccountPriceEndpoint,
		deleteAccountPriceEndpoint,
		exportPriceListEndpoint,
	}

	return &AccountPricesEndpointGroup{inner}
}
