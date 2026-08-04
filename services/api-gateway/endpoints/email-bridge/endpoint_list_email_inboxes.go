package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the account's email inboxes.
type ListEmailInboxesRequest struct{}

// Returns the account's email inboxes across every registered domain.
//
// Every inbox is returned in a single response; this list is not paginated.
type ListEmailInboxesEndpoint struct{}

func (e *ListEmailInboxesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEmailInboxesRequest, *apiresource.List[apiresource.EmailInbox]] {
	return (&apiendpoint.APIEndpoint[*ListEmailInboxesRequest, *apiresource.List[apiresource.EmailInbox]]{
		Title:               "List Email Inboxes",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-inboxes",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailInbox,
		IncludeConfig:       emailInboxIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEmailInboxesRequest) (*apiresource.List[apiresource.EmailInbox], *apierror.APIError) {
			return svc.(EmailBridgeSvc).ListInboxes
		},
	})
}
