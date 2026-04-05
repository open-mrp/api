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

	listEndpoint := (&accountintegrationep.ListAccountIntegrationsEndpoint{}).Materialize().WithService(inner, accountIntegrationSvc)
	createEndpoint := (&accountintegrationep.CreateAccountIntegrationEndpoint{}).Materialize().WithService(inner, accountIntegrationSvc)
	updateEndpoint := (&accountintegrationep.UpdateAccountIntegrationEndpoint{}).Materialize().WithService(inner, accountIntegrationSvc)
	deleteEndpoint := (&accountintegrationep.DeleteAccountIntegrationEndpoint{}).Materialize().WithService(inner, accountIntegrationSvc)
	getStripePublishableKeyEndpoint := (&accountintegrationep.GetStripePublishableKeyEndpoint{}).Materialize().WithService(inner, accountIntegrationSvc)
	getStripeStatusEndpoint := (&accountintegrationep.GetStripeStatusEndpoint{}).Materialize().WithService(inner, accountIntegrationSvc)

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
