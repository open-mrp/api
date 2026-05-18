package httpgroup

import (
	"fmt"

	accountintegrationep "github.com/augno/api/services/api-gateway/endpoints/account-integrations"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountIntegrationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountIntegrationsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountIntegrationsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account integrations endpoint group: core client is required")
	}
	return nil
}

func (*AccountIntegrationsEndpointGroup) Materialize(config *AccountIntegrationsEndpointGroupConfig) *AccountIntegrationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	accountIntegrationSvc := accountintegrationep.NewAccountIntegrationSvc(&accountintegrationep.AccountIntegrationSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Integrations",
		Description:  "List and manage third-party account integrations.",
		ResourceType: &apiresource.AccountIntegration{},
	}

	listEndpoint := apiendpoint.From(&accountintegrationep.ListAccountIntegrationsEndpoint{}).WithService(inner, accountIntegrationSvc)
	createEndpoint := apiendpoint.From(&accountintegrationep.CreateAccountIntegrationEndpoint{}).WithService(inner, accountIntegrationSvc)
	updateEndpoint := apiendpoint.From(&accountintegrationep.UpdateAccountIntegrationEndpoint{}).WithService(inner, accountIntegrationSvc)
	deleteEndpoint := apiendpoint.From(&accountintegrationep.DeleteAccountIntegrationEndpoint{}).WithService(inner, accountIntegrationSvc)
	getStripePublishableKeyEndpoint := apiendpoint.From(&accountintegrationep.GetStripePublishableKeyEndpoint{}).WithService(inner, accountIntegrationSvc)
	getStripeStatusEndpoint := apiendpoint.From(&accountintegrationep.GetStripeStatusEndpoint{}).WithService(inner, accountIntegrationSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		getStripePublishableKeyEndpoint,
		getStripeStatusEndpoint,
	}

	return &AccountIntegrationsEndpointGroup{inner}
}
