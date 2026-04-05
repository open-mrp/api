package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCarrierID = "cr_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleCarrierName = "FedEx"

// Carrier represents a shipping carrier configured for the account.
type Carrier struct {
	// The unique identifier for the carrier.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=carrier"`
	// The display name of the carrier.
	Name string `json:"name" validate:"required"`
	// The carrier code.
	Code *constants.CarrierCode `json:"code"`
	// The carrier account number, if applicable.
	AccountNumber *string `json:"account_number"`
	// Whether this carrier is visible in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required,enum"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// The service levels (shipping service levels).
	ServiceLevels []ServiceLevel `json:"service_levels" expandable:"true"`
	// When the carrier was soft-deleted, if applicable.
	DeletedAt *time.Time `json:"deleted_at"`
	// When the carrier was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the carrier was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCarrierCode constants.CarrierCode = constants.CarrierCodeFedEx

var SampleCarrier = &Carrier{
	ID:                       SampleCarrierID,
	Object:                   constants.ObjectTypeCarrier,
	Name:                     SampleCarrierName,
	Code:                     &sampleCarrierCode,
	CustomerPortalVisibility: constants.CustomerPortalVisibilityVisible,
	Owner:                    SampleOwnerAccount,
	CreatedAt:                timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:                timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Carrier) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCarrier)
}

// OAuthResponse represents the response from initiating carrier OAuth.
type OAuthResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=oauth_response"`
	// The OAuth URL to redirect the user to.
	OAuthURL string `json:"oauth_url" validate:"required"`
}

var SampleOAuthResponse = &OAuthResponse{
	Object:   constants.ObjectTypeOAuthResponse,
	OAuthURL: "https://oauth.fedex.com/authorize?client_id=abc123&redirect_uri=https://app.augno.com/carriers/oauth/callback",
}

func (*OAuthResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthResponse)
}

// OAuthStatusResponse represents the OAuth connection status for a carrier.
type OAuthStatusResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=oauth_status_response"`
	// The OAuth connection status ("connected" or "disconnected").
	Status string `json:"status" validate:"required"`
}

var SampleOAuthStatusResponse = &OAuthStatusResponse{
	Object: constants.ObjectTypeOAuthStatusResponse,
	Status: "connected",
}

func (*OAuthStatusResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthStatusResponse)
}
