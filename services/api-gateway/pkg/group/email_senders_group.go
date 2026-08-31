package httpgroup

import (
	"fmt"

	emailbridgeep "github.com/open-mrp/api/services/api-gateway/endpoints/email-bridge"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type EmailSendersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EmailSendersEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *EmailSendersEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("email senders endpoint group: notification client is required")
	}
	return nil
}

func (*EmailSendersEndpointGroup) Materialize(config *EmailSendersEndpointGroupConfig) *EmailSendersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	emailBridgeSvc := emailbridgeep.NewEmailBridgeSvc(&emailbridgeep.EmailBridgeSvcConfig{
		EmailBridgeClient: config.NotificationClient.EmailBridgeClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Email Senders",
		Description:  "Choose the address your order, invoice, and statement emails are sent from, on a domain you have verified.",
		ResourceType: &apiresource.EmailSender{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&emailbridgeep.GetEmailSenderEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.SetEmailSenderEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.DeleteEmailSenderEndpoint{}).WithService(inner, emailBridgeSvc),
	}

	return &EmailSendersEndpointGroup{inner}
}
