package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountIntegrationID = "ai_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleAccountIntegrationName = "My Stripe Integration"

// AccountIntegration represents a third-party integration connected to an account.
type AccountIntegration struct {
	// The unique identifier for the account integration.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_integration"`
	// The human-readable name for the integration.
	Name string `json:"name" validate:"required"`
	// The integration provider code (e.g. "stripe", "shippo").
	IntegrationCode constants.IntegrationCode `json:"integration_code" validate:"required"`
	// Whether this integration is currently active.
	IsActive bool `json:"is_active"`
	// When this integration was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this integration was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountIntegration = &AccountIntegration{
	ID:              SampleAccountIntegrationID,
	Object:          constants.ObjectTypeAccountIntegration,
	Name:            SampleAccountIntegrationName,
	IntegrationCode: constants.IntegrationCodeStripe,
	IsActive:        true,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountIntegration) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountIntegration)
}

// StripePublishableKey represents the Stripe publishable key for an account.
type StripePublishableKey struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=stripe_publishable_key"`
	// The Stripe publishable key.
	PublishableKey string `json:"publishable_key" validate:"required"`
}

var SampleStripePublishableKey = &StripePublishableKey{
	Object:         constants.ObjectTypeStripePublishableKey,
	PublishableKey: "pk_test_example123",
}

func (*StripePublishableKey) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleStripePublishableKey)
}

// StripeStatus represents whether an account has a Stripe integration.
type StripeStatus struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=stripe_status"`
	// Whether the account has a Stripe integration configured.
	HasStripeIntegration bool `json:"has_stripe_integration"`
}

var SampleStripeStatus = &StripeStatus{
	Object:               constants.ObjectTypeStripeStatus,
	HasStripeIntegration: true,
}

func (*StripeStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleStripeStatus)
}
