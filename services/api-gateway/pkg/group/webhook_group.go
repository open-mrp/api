package httpgroup

import (
	"fmt"

	webhooksep "github.com/augno/api/services/api-gateway/endpoints/webhooks"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type WebhooksEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type WebhooksEndpointGroupConfig struct {
	BillingClient *grpcclient.BillingServiceClient
}

func (c *WebhooksEndpointGroupConfig) validate() error {
	if c.BillingClient == nil {
		return fmt.Errorf("webhooks endpoint group: billing client is required")
	}
	return nil
}

func (*WebhooksEndpointGroup) Materialize(config *WebhooksEndpointGroupConfig) *WebhooksEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	webhookSvc := webhooksep.NewWebhookSvc(&webhooksep.WebhookSvcConfig{
		BillingClient: config.BillingClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Webhooks",
		Description:  "Handles incoming webhook events.",
		ResourceType: &apiresource.WebhookResponse{},
	}

	processWebhookEndpoint := (&webhooksep.ProcessWebhookEndpoint{}).Materialize().WithService(inner, webhookSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		processWebhookEndpoint,
	}

	return &WebhooksEndpointGroup{inner}
}
