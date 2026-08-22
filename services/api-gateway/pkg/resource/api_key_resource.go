package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAPIKeyID = "apke_eiylmwr6q7oz"                   // #nosec G101 - sample data for API docs
const SampleAPIKeyName = "Production API Key"                // #nosec G101 - sample data for API docs
const SampleTestAPIKeyRedactedValue = "mrp_sk_test_****kuIb" // #nosec G101 - sample data for API docs
const SampleProdAPIKeyRedactedValue = "mrp_sk_prod_****hjt4" // #nosec G101 - sample data for API docs

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleTestAPIKeyValue = "mrp_sk_test_RhxFDvTdDnb0bgtcoA5P79_60EmH4h9j9ZldsuU9XyngXlpu8NqdIlGTQw8OM8cGeCadykuIb"

// #nosec G101 - This is sample data for API documentation, not a real credential
const SampleProdAPIKeyValue = "mrp_sk_prod_RhxFDvTdDnb0bgtcoA5P79_60EmH4h9j9ZldsuU9XyngXlpu8NqdIlGTQw8OM8cGeCadyhjtr"

// An API key used to authenticate requests to the OpenMRP API.
//
// A key always acts on behalf of the account it was created under, with the permissions of the role assigned to it.
type APIKey struct {
	// API key ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=api_key"`
	// Human-readable name for the API key.
	Name string `json:"name" validate:"required"`
	// Redacted key value safe for display.
	//
	// The key's prefix followed by its last four characters, e.g. `mrp_sk_prod_****hjt4`.
	RedactedValue string `json:"redacted_value" validate:"required"`
	// Role assigned to the key, which determines the permissions of requests made with it.
	Role *Role `json:"role" expandable:"true"`
	// When the key was last used to authenticate a request.
	//
	// Recorded at most once every 24 hours, so it can lag the key's most recent use by up to a day.
	LastUsedAt *time.Time `json:"last_used_at"`
	// When the key expires and stops authenticating requests.
	//
	// A key with no expiration keeps working until it is revoked or rotated.
	ExpiresAt *time.Time `json:"expires_at"`
	// When the key's revocation takes effect.
	//
	// A future timestamp means revocation was scheduled (for example, by a rotation) and the key continues to authenticate requests until that time.
	RevokedAt *time.Time `json:"revoked_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A newly issued API key together with its secret value, returned when a key is created or rotated.
type CreatedAPIKey struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=created_api_key"`
	// The secret used to authenticate requests, sent as a bearer token in the `Authorization` header.
	//
	// This is the only response that ever contains the secret; if it is lost, rotate the key to issue a new one. Learn more about [managing your API keys](https://docs.augno.com/api/managing-api-keys).
	APIKeySecret string `json:"api_key_secret" validate:"required" sensitive:"true"`
	// The key's non-secret details, such as its ID, name, role, and expiration.
	APIKeyInfo APIKey `json:"api_key_info" validate:"required"`
}

var SampleAPIKey = &APIKey{
	ID:            SampleAPIKeyID,
	Object:        constants.ObjectTypeAPIKey,
	Name:          SampleAPIKeyName,
	RedactedValue: SampleProdAPIKeyRedactedValue,
	Role:          SampleRole,
	LastUsedAt:    timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	ExpiresAt:     timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	RevokedAt:     nil,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
