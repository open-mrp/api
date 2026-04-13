package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get carrier OAuth status.
type GetOAuthStatusRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

type GetOAuthStatusEndpoint struct{}

func (e *GetOAuthStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetOAuthStatusRequest, *apiresource.OAuthStatusResponse] {
	return &apiendpoint.APIEndpoint[*GetOAuthStatusRequest, *apiresource.OAuthStatusResponse]{
		Title:             "Get Carrier OAuth Status",
		Description:       "Returns the OAuth connection status for a carrier. Sandbox accounts always return disconnected.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{id}/oauth-status",
		Request:           &GetOAuthStatusRequest{},
		Response:          &apiresource.OAuthStatusResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetOAuthStatusRequest) (*apiresource.OAuthStatusResponse, *apierror.APIError) {
			return svc.(CarrierSvc).GetOAuthStatus
		},
	}
}
