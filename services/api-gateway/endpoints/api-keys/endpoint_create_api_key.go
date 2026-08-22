package apikeyep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create an API key.
type CreateAPIKeyRequest struct {
	// ID of the role to assign to the API key.
	//
	// The role determines what requests authenticated with the key are allowed to do. A key keeps its role for life — including through rotation — so issue a new key to use a different one, while changes to the role's own permissions take effect for existing keys immediately.
	RoleID string `json:"role_id" validate:"required"`
	// Human-readable name for the API key.
	//
	// Shown when listing keys and used to match keys when searching, so prefer something that identifies the integration using it.
	Name string `json:"name" validate:"required,max=255"`
	// When the key expires and stops authenticating requests.
	//
	// If omitted, the key keeps working until it is revoked or rotated.
	ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
}

var sampleCreateAPIKeyExpiresAt = time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
var sampleCreateAPIKeyRequest = &CreateAPIKeyRequest{
	RoleID:    apiresource.SampleRoleID,
	Name:      apiresource.SampleAPIKeyName,
	ExpiresAt: field.Some(sampleCreateAPIKeyExpiresAt),
}

func (*CreateAPIKeyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAPIKeyRequest)
}

// Creates an [API key](https://docs.augno.com/api/api-keys) to authenticate API requests.
//
// The key belongs to the account it was created under and only ever acts on behalf of that account. Keys created under a sandbox account carry an `mrp_sk_test_` prefix; keys created under a production account carry an `mrp_sk_prod_` prefix.
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
		AgentTool:         false,
		RequiredRoleType:  constants.RoleTypeAdmin,
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
