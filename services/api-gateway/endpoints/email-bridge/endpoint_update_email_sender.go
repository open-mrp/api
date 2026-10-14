package emailbridgeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update the address the account's customer-facing email is sent from.
type UpdateEmailSenderRequest struct {
	// The verified email domain to send from. Required when no sender is set up yet.
	EmailDomainID field.Optional[string] `json:"email_domain_id,omitzero"`
	// The mailbox name before the `@`, such as `orders`. Required when no sender is set up yet.
	LocalPart field.Optional[string] `json:"local_part,omitzero"`
	// The name shown in a mail client's sender column. Send `null` to show the bare address.
	FromName field.Clearable[string] `json:"from_name,omitzero"`
	// Where customer replies are delivered. Send `null` to deliver replies to the sending address.
	ReplyTo field.Clearable[string] `json:"reply_to,omitzero"`
}

var sampleUpdateEmailSenderRequest = &UpdateEmailSenderRequest{
	EmailDomainID: field.Some(apiresource.SampleEmailDomainID),
	LocalPart:     field.Some("orders"),
	FromName:      field.Set("Acme Inc."),
}

func (*UpdateEmailSenderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateEmailSenderRequest)
}

// Updates your email sender, setting it up if you don't have one.
//
// Only the fields you send change. The domain must be verified. Emails about someone's OpenMRP account, such as password resets, still send from the platform address.
type UpdateEmailSenderEndpoint struct{}

func (e *UpdateEmailSenderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateEmailSenderRequest, *apiresource.EmailSender] {
	return (&apiendpoint.APIEndpoint[*UpdateEmailSenderRequest, *apiresource.EmailSender]{
		Title:             "Update Email Sender",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-sender",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEmailSender,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		IncludeConfig:     emailSenderIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateEmailSenderRequest) (*apiresource.EmailSender, *apierror.APIError) {
			return svc.(EmailBridgeSvc).UpdateSender
		},
	})
}
