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

// Request to delete an email domain.
type DeleteEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Deregisters a domain from the email bridge and removes its sending identity from the mail provider.
//
// Delete the domain's inboxes first: while any inbox still exists on it, this returns a conflict error.
type DeleteEmailDomainEndpoint struct{}

func (e *DeleteEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteEmailDomainRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteEmailDomainRequest, *object.Deleted]{
		Title:             "Delete Email Domain",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-domains/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEmailDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteEmailDomainRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(EmailBridgeSvc).DeleteDomain, constants.ObjectTypeEmailDomain, "id")
		},
	})
}
