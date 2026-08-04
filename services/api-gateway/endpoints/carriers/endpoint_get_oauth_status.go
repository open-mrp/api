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

// Reports whether a carrier's account has been authorized for live rating and label purchase.
//
// Only carriers connected through Shippo have an account to authorize; every other carrier reports `disconnected`, as do sandbox accounts and carriers whose account cannot be reached.
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
