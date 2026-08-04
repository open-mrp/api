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

// Request to read the support route configured for one scope.
type GetSupportRouteRequest struct {
	// The customer account whose override to read.
	//
	// Omit to read the account-level default instead.
	RelationAccountID *string `query:"relation_account_id"`
}

// Retrieves the support route configured for one scope in your account.
//
// This reads the exact scope you ask for and does not fall back: asking for a customer that has no override of its own returns a not-found error even when an account-level default is configured, so a caller checking which route will actually be used for a customer must also read the default.
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
