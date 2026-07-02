package httpgroup

import (
	"fmt"

	emailbridgeep "github.com/augno/api/services/api-gateway/endpoints/email-bridge"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type EmailDomainsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EmailDomainsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *EmailDomainsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("email domains endpoint group: notification client is required")
	}
	return nil
}

func (*EmailDomainsEndpointGroup) Materialize(config *EmailDomainsEndpointGroupConfig) *EmailDomainsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	emailBridgeSvc := emailbridgeep.NewEmailBridgeSvc(&emailbridgeep.EmailBridgeSvcConfig{
		EmailBridgeClient: config.NotificationClient.EmailBridgeClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Email Domains",
		Description:  "Register customer-owned domains with the email bridge and verify them for sending and receiving mail.",
		ResourceType: &apiresource.EmailDomain{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&emailbridgeep.CreateEmailDomainEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.ListEmailDomainsEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.GetEmailDomainEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.VerifyEmailDomainEndpoint{}).WithService(inner, emailBridgeSvc),
	}

	return &EmailDomainsEndpointGroup{inner}
}
