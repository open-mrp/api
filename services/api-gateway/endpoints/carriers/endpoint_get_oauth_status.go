package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get carrier OAuth status.
type GetOAuthStatusRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

// Returns the OAuth connection status for a carrier.
//
// The status is one of `connected`, `authorization_pending`, or `disconnected`. Sandbox accounts always return `disconnected`.
type GetOAuthStatusEndpoint struct{}

func (e *GetOAuthStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetOAuthStatusRequest, *apiresource.OAuthStatusResponse] {
	return (&apiendpoint.APIEndpoint[*GetOAuthStatusRequest, *apiresource.OAuthStatusResponse]{
		Title:               "Get Carrier OAuth Status",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/carriers/{id}/oauth-status",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionRead}},
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetOAuthStatusRequest) (*apiresource.OAuthStatusResponse, *apierror.APIError) {
			return svc.(CarrierSvc).GetOAuthStatus
		},
	})
}
