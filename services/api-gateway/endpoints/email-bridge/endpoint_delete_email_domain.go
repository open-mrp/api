package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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

func (e *DeleteEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteEmailDomainRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteEmailDomainRequest, *apiresource.EmptyResource]{
		Title:               "Delete Email Domain",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-domains/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteEmailDomainRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(EmailBridgeSvc).DeleteDomain
		},
	})
}
