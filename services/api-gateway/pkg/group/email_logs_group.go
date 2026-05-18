package httpgroup

import (
	"fmt"

	emaillogep "github.com/augno/api/services/api-gateway/endpoints/email-logs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type EmailLogsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EmailLogsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *EmailLogsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("email logs endpoint group: core client is required")
	}
	return nil
}

func (*EmailLogsEndpointGroup) Materialize(config *EmailLogsEndpointGroupConfig) *EmailLogsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	emailLogSvc := emaillogep.NewEmailLogSvc(&emaillogep.EmailLogSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Email Logs",
		Description:  "View email logs for accounts.",
		ResourceType: &apiresource.EmailLog{},
	}

	listEmailLogsEndpoint := apiendpoint.From(&emaillogep.ListEmailLogsEndpoint{}).WithService(inner, emailLogSvc)
	getEmailLogEndpoint := apiendpoint.From(&emaillogep.RetrieveEmailLogEndpoint{}).WithService(inner, emailLogSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEmailLogsEndpoint,
		getEmailLogEndpoint,
	}

	return &EmailLogsEndpointGroup{inner}
}
