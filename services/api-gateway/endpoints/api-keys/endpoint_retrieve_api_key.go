package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get an API key by ID.
type RetrieveAPIKeyRequest struct {
	// API key ID.
	APIKeyID string `path:"id" validate:"required"`
}

// Returns API key metadata by ID.
type RetrieveAPIKeyEndpoint struct{}

func (e *RetrieveAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAPIKeyRequest, *apiresource.APIKey] {
	return (&apiendpoint.APIEndpoint[*RetrieveAPIKeyRequest, *apiresource.APIKey]{
		Title:             "Retrieve API Key",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/auth/api-keys/{id}",
		Request:           &RetrieveAPIKeyRequest{},
		Response:          &apiresource.APIKey{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError) {
			return svc.(APIKeySvc).GetAPIKey
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
		}),
	}).WithDocSource(e)
}
