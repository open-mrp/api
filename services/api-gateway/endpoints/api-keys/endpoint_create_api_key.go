package apikeyep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create an API key.
type CreateAPIKeyRequest struct {
	// Role ID assigned to the API key.
	RoleID string `json:"role_id" validate:"required"`
	// Human-readable name for the API key.
	Name string `json:"name" validate:"required,max=255"`
	// Expiration timestamp. If not set, the key does not expire.
	ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
}

var sampleCreateAPIKeyRequest = &CreateAPIKeyRequest{
	RoleID: apiresource.SampleRoleID,
	Name:   apiresource.SampleAPIKeyName,
}

func (*CreateAPIKeyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAPIKeyRequest)
}

// Creates an [API key](https://docs.augno.com/api/api-keys) to authenticate API requests.
//
// The secret key is returned once and cannot be retrieved later, so you should store it securely. We provide some [recommendations](https://docs.augno.com/api/managing-api-keys) on how you can manage your API keys.
type CreateAPIKeyEndpoint struct{}

func (e *CreateAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *apiresource.CreatedAPIKey] {
	return (&apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *apiresource.CreatedAPIKey]{
		Title:             "Create API Key",
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCreatedAPIKey,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
			return svc.(APIKeySvc).CreateAPIKey
		},
		LocationFunc: func(resp *apiresource.CreatedAPIKey) string {
			return "/v1/auth/api-keys/" + resp.APIKeyInfo.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
			PathPrefix: "api_key_info",
		}),
	})
}
