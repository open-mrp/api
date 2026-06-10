package httpgroup

import (
	"fmt"

	accountstatusep "github.com/augno/api/services/api-gateway/endpoints/account-statuses"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AccountStatusesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AccountStatusesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AccountStatusesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account statuses endpoint group: core client is required")
	}
	return nil
}

func (*AccountStatusesEndpointGroup) Materialize(config *AccountStatusesEndpointGroupConfig) *AccountStatusesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	accountStatusSvc := accountstatusep.NewAccountStatusSvc(&accountstatusep.AccountStatusSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Account Statuses",
		Description:  "List and retrieve account statuses.",
		ResourceType: &apiresource.AccountStatus{},
	}

	listAccountStatusesEndpoint := apiendpoint.From(&accountstatusep.ListAccountStatusesEndpoint{}).WithService(inner, accountStatusSvc)
	getAccountStatusEndpoint := apiendpoint.From(&accountstatusep.RetrieveAccountStatusEndpoint{}).WithService(inner, accountStatusSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listAccountStatusesEndpoint,
		getAccountStatusEndpoint,
	}

	return &AccountStatusesEndpointGroup{inner}
}
