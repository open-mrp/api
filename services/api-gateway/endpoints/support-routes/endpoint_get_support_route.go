package supportrouteep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to read the support route for a scope.
type GetSupportRouteRequest struct {
	// The customer account whose override to read.
	//
	// Omit to read the account-level default instead.
	RelationAccountID *string `query:"relation_account_id"`
}

// Returns the support route for an exact scope in the caller's account.
type GetSupportRouteEndpoint struct{}

func (e *GetSupportRouteEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSupportRouteRequest, *apiresource.SupportRoute] {
	return (&apiendpoint.APIEndpoint[*GetSupportRouteRequest, *apiresource.SupportRoute]{
		Title:               "Get Support Route",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/support-routes",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeSupportRoute,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSupportRouteRequest) (*apiresource.SupportRoute, *apierror.APIError) {
			return svc.(SupportRouteSvc).GetSupportRoute
		},
	})
}
