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
	// Redirect URI after OAuth completes.
	RedirectURI string `json:"redirect_uri" validate:"required"`
	// Opaque state value passed through the OAuth flow.
	State field.Optional[string] `json:"state,omitzero"`
}

var sampleInitiateOAuthRequest = &InitiateOAuthRequest{
	RedirectURI: "https://app.example.com/carriers/oauth/callback",
}

func (*InitiateOAuthRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleInitiateOAuthRequest)
}

// Initiates the OAuth authorization flow for a Shippo-managed carrier and returns the URL to redirect the user to.
//
// Not available in sandbox mode.
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
