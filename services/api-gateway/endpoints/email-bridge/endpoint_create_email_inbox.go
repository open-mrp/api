package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to provision a routable inbox on a verified domain.
type CreateEmailInboxRequest struct {
	// The verified domain this inbox belongs to.
	EmailDomainID string `json:"email_domain_id" validate:"required"`
	// The full inbox address (e.g. `support@acme.com`).
	//
	// Its domain part must match the selected domain, which must already be verified.
	Address string `json:"address" validate:"required"`
	// Display name for the `From` header of outbound mail.
	FromName field.Optional[string] `json:"from_name,omitzero"`
	// The agent to bind to this inbox to handle incoming mail.
	AgentConfigID field.Optional[string] `json:"agent_config_id,omitzero"`
	// How the bound agent decides whether to run on incoming mail.
	//
	// - `mention`: runs only when the agent is @mentioned in the message.
	// - `keyword`: runs when the message contains any of the trigger keywords.
	// - `always`: runs on every incoming message.
	AgentTriggerPolicy field.Optional[string] `json:"agent_trigger_policy,omitzero"`
	// Keywords that fire the agent when the trigger policy is `keyword`.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords,omitzero"`
}

var sampleCreateEmailInboxRequest = &CreateEmailInboxRequest{
	EmailDomainID: apiresource.SampleEmailDomainID,
	Address:       "support@acme.com",
}

func (*CreateEmailInboxRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateEmailInboxRequest)
}

// Provisions a routable inbox address on a verified domain.
type CreateEmailInboxEndpoint struct{}

func (e *CreateEmailInboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateEmailInboxRequest, *apiresource.EmailInbox] {
	return (&apiendpoint.APIEndpoint[*CreateEmailInboxRequest, *apiresource.EmailInbox]{
		Title:               "Create Email Inbox",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-inboxes",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailInbox,
		IncludeConfig:       emailInboxIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
			return svc.(EmailBridgeSvc).CreateInbox
		},
	})
}
