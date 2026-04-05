package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// RevokeAPIKeyRequest is the request to revoke an API key.
type RevokeAPIKeyRequest struct {
	// The ID of the API key to revoke.
	APIKeyID string `path:"id" validate:"required"`
}

type RevokeAPIKeyEndpoint struct{}

func (e *RevokeAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *apiresource.EmptyResource]{
		Title:             "Revoke API Key",
		Description:       "Revokes an API key so it can no longer be used to authenticate requests.",
		Method:            http.MethodDelete,
		Route:             "/v1/auth/api-keys/{id}",
		ContentType:       "application/json",
		Request:           &RevokeAPIKeyRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RevokeAPIKeyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(APIKeySvc).RevokeAPIKey
		},
	}
}
