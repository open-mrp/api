package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to configure the address the account's customer-facing email is sent from.
type SetEmailSenderRequest struct {
	// The verified email domain to send from.
	EmailDomainID string `json:"email_domain_id" validate:"required"`
	// The mailbox name before the `@`, for example `orders`.
	LocalPart string `json:"local_part" validate:"required"`
	// The name shown in a mail client's sender column. When unset, mail shows the bare address.
	FromName field.Optional[string] `json:"from_name,omitzero"`
	// Where customer replies are delivered. When unset, replies go to the sending address.
	ReplyTo field.Optional[string] `json:"reply_to,omitzero"`
}

var sampleSetEmailSenderFromName = "Acme Inc."

var sampleSetEmailSenderRequest = &SetEmailSenderRequest{
	EmailDomainID: apiresource.SampleEmailDomainID,
	LocalPart:     "orders",
	FromName:      field.Some(sampleSetEmailSenderFromName),
}

func (*SetEmailSenderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetEmailSenderRequest)
}

// Sets the address your order, invoice, and statement emails are sent from, replacing any address already configured.
//
// The domain must be verified first. Emails about someone's OpenMRP account — password resets, verification, plan changes — continue to send from the platform address.
type SetEmailSenderEndpoint struct{}

func (e *SetEmailSenderEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetEmailSenderRequest, *apiresource.EmailSender] {
	return (&apiendpoint.APIEndpoint[*SetEmailSenderRequest, *apiresource.EmailSender]{
		Title:               "Set Email Sender",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-sender",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailSender,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetEmailSenderRequest) (*apiresource.EmailSender, *apierror.APIError) {
			return svc.(EmailBridgeSvc).SetSender
		},
	})
}
