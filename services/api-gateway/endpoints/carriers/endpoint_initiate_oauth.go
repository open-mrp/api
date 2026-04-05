package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// InitiateOAuthRequest is the request to initiate carrier OAuth.
type InitiateOAuthRequest struct {
	// The ID of the carrier.
	CarrierID string `path:"id" validate:"required"`
	// The URI to redirect to after OAuth completes.
	RedirectURI string `json:"redirect_uri" validate:"required"`
	// An optional opaque state value passed through the OAuth flow.
	State *string `json:"state"`
}

var sampleInitiateOAuthRequest = &InitiateOAuthRequest{
	RedirectURI: "https://app.example.com/carriers/oauth/callback",
}

func (*InitiateOAuthRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleInitiateOAuthRequest)
}

type InitiateOAuthEndpoint struct{}

func (e *InitiateOAuthEndpoint) Materialize() *apiendpoint.APIEndpoint[*InitiateOAuthRequest, *apiresource.OAuthResponse] {
	return &apiendpoint.APIEndpoint[*InitiateOAuthRequest, *apiresource.OAuthResponse]{
		Title:             "Initiate Carrier OAuth",
		Description:       "Initiates the OAuth flow for a Shippo-managed carrier and returns an OAuth URL to redirect the user to. Not available in sandbox mode.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/carriers/{id}/actions/initiate-oauth",
		Request:           &InitiateOAuthRequest{},
		Response:          &apiresource.OAuthResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *InitiateOAuthRequest) (*apiresource.OAuthResponse, *apierror.APIError) {
			return svc.(CarrierSvc).InitiateOAuth
		},
	}
}
