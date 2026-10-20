package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// Acknowledgement that a webhook event was accepted.
type WebhookResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=webhook_response"`
	// Whether the event was accepted for processing.
	//
	// Acceptance means the signature was verified and the event was handled or queued. Event types OpenMRP takes no action on are acknowledged the same way, so this is not a signal that anything changed.
	Received bool `json:"received"`
}

var SampleWebhookResponse = &WebhookResponse{
	Object:   constants.ObjectTypeWebhookResponse,
	Received: true,
}

func (*WebhookResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleWebhookResponse)
}
