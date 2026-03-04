package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAPIKeyID = "apke_01jm4r6700e3kxb9w2nqh7g5fp"    // #nosec G101 - sample data for API docs
const SampleAPIKeyName = "Production API Key"               // #nosec G101 - sample data for API docs
const SampleTestAPIKeyRedactedValue = "aug_sk_test_...kuIb" // #nosec G101 - sample data for API docs
const SampleProdAPIKeyRedactedValue = "aug_sk_prod_...hjt4" // #nosec G101 - sample data for API docs

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleTestAPIKeyValue = "aug_sk_test_RhxFDvTdDnb0bgtcoA5P79_60EmH4h9j9ZldsuU9XyngXlpu8NqdIlGTQw8OM8cGeCadykuIb"

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleProdAPIKeyValue = "aug_sk_prod_RhxFDvTdDnb0bgtcoA5P79_60EmH4h9j9ZldsuU9XyngXlpu8NqdIlGTQw8OM8cGeCadyhjtr"

var SampleAPIKey = &APIKey{
	ID:            SampleAPIKeyID,
	Object:        constants.ObjectTypeAPIKey,
	Name:          SampleAPIKeyName,
	RedactedValue: SampleProdAPIKeyRedactedValue,
	Role: &LightRole{
		ID:           SampleRoleID,
		Object:       constants.ObjectTypeRole,
		Name:         SampleRoleName,
		RoleTypeCode: constants.RoleTypeCodeAdmin,
	},
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	LastUsedAt: timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	ExpiresAt:  timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	RevokedAt:  nil,
}

var SampleCreatedAPIKey = &CreatedAPIKey{
	APIKeySecret: SampleProdAPIKeyValue,
	APIKeyInfo:   *SampleAPIKey,
}

// APIKey represents an API key for authenticating API requests.
type APIKey struct {
	// The unique identifier for the API key.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=api_key"`
	// The human-readable name for the API key.
	Name string `json:"name" validate:"required"`
	// The redacted value of the API key for display purposes.
	RedactedValue string `json:"redacted_value" validate:"required"`
	// The role associated with this API key. Expandable.
	Role *LightRole `json:"role" expandable:"true"`
	// The timestamp when the API key was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the API key was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
	// The timestamp when the API key was last used.
	LastUsedAt *time.Time `json:"last_used_at"`
	// The timestamp when the API key expires.
	ExpiresAt *time.Time `json:"expires_at"`
	// The timestamp when the API key was revoked.
	RevokedAt *time.Time `json:"revoked_at"`
}

func (*APIKey) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAPIKey)
}

// CreatedAPIKey represents a newly created API key with the full secret value.
type CreatedAPIKey struct {
	// The full API key secret value (only shown once at creation).
	APIKeySecret string `json:"api_key_secret" validate:"required"`
	// The API key metadata.
	APIKeyInfo APIKey `json:"api_key_info" validate:"required"`
}

func (*CreatedAPIKey) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreatedAPIKey)
}
