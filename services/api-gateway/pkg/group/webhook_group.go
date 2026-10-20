package httpgroup

import (
	"fmt"

	webhooksep "github.com/open-mrp/api/services/api-gateway/endpoints/webhooks"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type WebhooksEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type WebhooksEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client, used to verify and record per-account Stripe webhook events.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *WebhooksEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("webhooks endpoint group: core client is required")
	}
	return nil
}

func (*WebhooksEndpointGroup) Materialize(config *WebhooksEndpointGroupConfig) *WebhooksEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	webhookSvc := webhooksep.NewWebhookSvc(&webhooksep.WebhookSvcConfig{
		SalesClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Webhooks",
		Description:  "Incoming webhook events.",
		ResourceType: &apiresource.WebhookResponse{},
	}

	processAccountWebhookEndpoint := apiendpoint.From(&webhooksep.ProcessAccountWebhookEndpoint{}).WithService(inner, webhookSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		processAccountWebhookEndpoint,
	}

	return &WebhooksEndpointGroup{inner}
}
