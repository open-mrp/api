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
)

// The request to create an API key
type CreateAPIKeyRequest struct {
	// The role ID for the API key.
	RoleID string `json:"role_id" validate:"required,max=191"`
	// The name for the API key.
	Name string `json:"name" validate:"required,max=255"`
	// Optional expiration time for the API key.
	ExpiresAt *time.Time `json:"expires_at,omitempty" nullable:"false"`
}

var sampleCreateAPIKeyRequest = &CreateAPIKeyRequest{
	RoleID: apiresource.SampleRoleID,
	Name:   apiresource.SampleAPIKeyName,
}

func (*CreateAPIKeyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAPIKeyRequest)
}

type CreateAPIKeyEndpoint struct{}

func (e *CreateAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *apiresource.CreatedAPIKey] {
	return &apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *apiresource.CreatedAPIKey]{
		Title:             "Create API Key",
		Description:       "Creates a new API key. The secret value is returned only once at creation and cannot be retrieved afterward.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys",
		ContentType:       "application/json",
		Request:           &CreateAPIKeyRequest{},
		Response:          &apiresource.CreatedAPIKey{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
		Extras: apiendpoint.APIEndpointExtras{
			ShieldResponseBody: true,
		},
	}
}
