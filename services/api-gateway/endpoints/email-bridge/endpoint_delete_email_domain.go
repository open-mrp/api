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

// Request to delete an email domain.
type DeleteEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Deregisters a customer-owned domain from the email bridge.
//
// The domain's SES identity is removed. The domain must have no inboxes bound to it.
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
