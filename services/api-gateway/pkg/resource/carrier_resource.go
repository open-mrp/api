package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleCarrierID = "cr_01784fd54c9ba197bb4e42f0e6"
const SampleCarrierName = "FedEx"

// A shipping carrier configured for fulfilling orders.
//
// Carriers with a Shippo-supported `code` (`fedex`, `ups`, `usps`) are connected through Shippo for live rating and label purchase; other carriers represent self-managed shipping methods such as will call or local delivery.
type Carrier struct {
	// Carrier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=carrier"`
	// Human-readable name for the carrier, unique among your account's carriers.
	Name string `json:"name" validate:"required"`
	// Well-known carrier identifier, set only for recognized carriers and absent for custom ones.
	//
	// - `fedex`, `ups`, `usps`: integrated carriers managed through Shippo (live rating and labels).
	// - `will_call`: customer picks the order up; no carrier shipment.
	// - `delivery`: delivered by your own vehicles/drivers.
	// - `ltl`, `ltl1`: less-than-truckload freight carriers.
	// - `freight_collect`: freight billed to and arranged by the receiver.
	Code *constants.CarrierCode `json:"code"`
	// Your account number with this carrier, used to connect UPS and USPS accounts.
	AccountNumber *string `json:"account_number"`
	// Whether customers can see and select this carrier at checkout in the customer portal.
	CustomerPortalVisibility constants.CustomerPortalVisibility `json:"customer_portal_visibility" validate:"required"`
	// Provenance of this carrier.
	//
	// System-owned carriers are platform-provided defaults shared across all accounts and cannot be deleted; account-owned carriers are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Shipping service levels offered by this carrier (e.g. ground, overnight).
	ServiceLevels *List[ServiceLevel] `json:"service_levels" expandable:"true"`
	// Soft-delete timestamp.
	DeletedAt *time.Time `json:"deleted_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCarrierCode constants.CarrierCode = constants.CarrierCodeFedEx

var sampleCarrierAccountNumber = "603145678"

var SampleCarrier = &Carrier{
	ID:                       SampleCarrierID,
	Object:                   constants.ObjectTypeCarrier,
	Name:                     SampleCarrierName,
	Code:                     &sampleCarrierCode,
	AccountNumber:            &sampleCarrierAccountNumber,
	CustomerPortalVisibility: constants.CustomerPortalVisibilityVisible,
	Owner:                    SampleOwnerAccount,
	ServiceLevels:            NewList([]ServiceLevel{*SampleServiceLevel}, PageInfo{}),
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
	OAuthURL: "https://oauth.fedex.com/authorize?client_id=abc123&redirect_uri=https://www.augno.com/carriers/oauth/callback",
}

func (*OAuthResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthResponse)
}

// OAuth connection status for a carrier.
type OAuthStatusResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=oauth_status_response"`
	// OAuth connection status.
	//
	// - `connected`: the carrier account is authorized and ready for live rating and labels.
	// - `authorization_pending`: the carrier account exists but OAuth authorization has not been completed.
	// - `disconnected`: no carrier account is connected. Sandbox accounts always report this status.
	Status string `json:"status" validate:"required"`
}

var SampleOAuthStatusResponse = &OAuthStatusResponse{
	Object: constants.ObjectTypeOAuthStatusResponse,
	Status: "connected",
}

func (*OAuthStatusResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthStatusResponse)
}
