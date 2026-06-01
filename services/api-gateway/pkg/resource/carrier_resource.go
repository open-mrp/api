package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCarrierID = "cr_01784fd54c9ba197bb4e42f0e6"
const SampleCarrierName = "FedEx"

// Carrier resource.
type Carrier struct {
	// Carrier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=carrier"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Carrier code.
	Code *constants.CarrierCode `json:"code"`
	// Account number.
	AccountNumber *string `json:"account_number"`
	// Customer portal visibility.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Service levels.
	ServiceLevels *List[ServiceLevel] `json:"service_levels" expandable:"true"`
	// Soft-delete timestamp.
	DeletedAt *time.Time `json:"deleted_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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

// Response from initiating carrier OAuth.
type OAuthResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=oauth_response"`
	// OAuth URL to redirect the user to.
	OAuthURL string `json:"oauth_url" validate:"required"`
}

var SampleOAuthResponse = &OAuthResponse{
	Object:   constants.ObjectTypeOAuthResponse,
	OAuthURL: "https://oauth.fedex.com/authorize?client_id=abc123&redirect_uri=https://app.augno.com/carriers/oauth/callback",
}

func (*OAuthResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthResponse)
}

// OAuth connection status for a carrier.
type OAuthStatusResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=oauth_status_response"`
	// OAuth connection status. One of "connected", "authorization_pending", or "disconnected".
	Status string `json:"status" validate:"required"`
}

var SampleOAuthStatusResponse = &OAuthStatusResponse{
	Object: constants.ObjectTypeOAuthStatusResponse,
	Status: "connected",
}

func (*OAuthStatusResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthStatusResponse)
}
