package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAPIKeyID = "apke_01fba3a7db3996e3b3b1a07e00"     // #nosec G101 - sample data for API docs
const SampleAPIKeyName = "Production API Key"                // #nosec G101 - sample data for API docs
const SampleTestAPIKeyRedactedValue = "aug_sk_test_****kuIb" // #nosec G101 - sample data for API docs
const SampleProdAPIKeyRedactedValue = "aug_sk_prod_****hjt4" // #nosec G101 - sample data for API docs

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleTestAPIKeyValue = "aug_sk_test_RhxFDvTdDnb0bgtcoA5P79_60EmH4h9j9ZldsuU9XyngXlpu8NqdIlGTQw8OM8cGeCadykuIb"

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleProdAPIKeyValue = "aug_sk_prod_RhxFDvTdDnb0bgtcoA5P79_60EmH4h9j9ZldsuU9XyngXlpu8NqdIlGTQw8OM8cGeCadyhjtr"

// API key resource.
type APIKey struct {
	// API key ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=api_key"`
	// Human-readable name for the API key.
	Name string `json:"name" validate:"required"`
	// Redacted key value safe for display.
	RedactedValue string `json:"redacted_value" validate:"required"`
	// Role assigned to the key, which determines the permissions of requests made with it.
	Role *Role `json:"role" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
	// When the key was last used to authenticate a request.
	//
	// `null` if it has never been used.
	LastUsedAt *time.Time `json:"last_used_at"`
	// When the key expires and stops authenticating.
	//
	// `null` if the key never expires.
	ExpiresAt *time.Time `json:"expires_at"`
	// When the key was revoked.
	//
	// `null` if the key has not been revoked.
	RevokedAt *time.Time `json:"revoked_at"`
}

// Result of creating an API key, with the full secret value.
type CreatedAPIKey struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=created_api_key"`
	// Full secret value.
	//
	// Returned once and cannot be retrieved later. Learn more about [managing your API keys](https://docs.augno.com/api/managing-api-keys).
	APIKeySecret string `json:"api_key_secret" validate:"required" sensitive:"true"`
	// API key metadata.
	APIKeyInfo APIKey `json:"api_key_info" validate:"required"`
}

var SampleAPIKey = &APIKey{
	ID:            SampleAPIKeyID,
	Object:        constants.ObjectTypeAPIKey,
	Name:          SampleAPIKeyName,
	RedactedValue: SampleProdAPIKeyRedactedValue,
	Role:          SampleRole,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	LastUsedAt:    timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	ExpiresAt:     timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	RevokedAt:     nil,
}

var SampleCreatedAPIKey = &CreatedAPIKey{
	Object:       constants.ObjectTypeCreatedAPIKey,
	APIKeySecret: SampleProdAPIKeyValue,
	APIKeyInfo:   *SampleAPIKey,
}

func (*APIKey) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAPIKey)
}

func (*CreatedAPIKey) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreatedAPIKey)
}
