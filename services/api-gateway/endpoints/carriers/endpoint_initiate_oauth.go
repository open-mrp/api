package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to initiate carrier OAuth.
type InitiateOAuthRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
	// URL the carrier sends the user back to once they finish authorizing.
	RedirectURI string `json:"redirect_uri" validate:"required"`
	// Opaque value passed through the OAuth flow and handed back on the redirect.
	//
	// Use it to correlate the callback with the request that started it, or to carry the page the user should return to.
	State field.Optional[string] `json:"state,omitzero"`
}

var sampleInitiateOAuthRequest = &InitiateOAuthRequest{
	RedirectURI: "https://app.example.com/carriers/oauth/callback",
}

func (*InitiateOAuthRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleInitiateOAuthRequest)
}

// Starts the OAuth flow that authorizes your own account with the carrier, returning the URL to send the user to.
//
// The carrier must already have a Shippo carrier account, which is created when the carrier is created with a Shippo-supported code. Not available in sandbox mode.
type InitiateOAuthEndpoint struct{}

func (e *InitiateOAuthEndpoint) Materialize() *apiendpoint.APIEndpoint[*InitiateOAuthRequest, *apiresource.OAuthResponse] {
	return (&apiendpoint.APIEndpoint[*InitiateOAuthRequest, *apiresource.OAuthResponse]{
		Title:               "Initiate Carrier OAuth",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/carriers/{id}/actions/initiate-oauth",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *InitiateOAuthRequest) (*apiresource.OAuthResponse, *apierror.APIError) {
			return svc.(CarrierSvc).InitiateOAuth
		},
	})
}
