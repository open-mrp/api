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
	"github.com/open-mrp/apikit/object"
)

// Request to list the account's email inboxes.
type ListEmailInboxesRequest struct {
	object.ListRequest
}

// Returns a list of your email inboxes across every domain.
//
// The inboxes are sorted by creation date, newest first.
type ListEmailInboxesEndpoint struct{}

func (e *ListEmailInboxesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEmailInboxesRequest, *apiresource.List[apiresource.EmailInbox]] {
	return (&apiendpoint.APIEndpoint[*ListEmailInboxesRequest, *apiresource.List[apiresource.EmailInbox]]{
		Title:             "List Email Inboxes",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-inboxes",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeEmailInbox,
		IncludeConfig:     emailInboxIncludeConfig(),
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEmailInboxesRequest) (*apiresource.List[apiresource.EmailInbox], *apierror.APIError) {
			return svc.(EmailBridgeSvc).ListInboxes
		},
	})
}
