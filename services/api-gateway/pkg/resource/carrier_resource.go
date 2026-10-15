package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleCarrierID = "cr_tv5vfjtgu1n3"
const SampleCarrierName = "FedEx"

// A carrier that ships orders, such as FedEx, an LTL freight line or your own delivery vans.
//
// Carriers your account created sit beside the system carriers every account shares.
type Carrier struct {
	// Carrier ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=carrier"`
	// Display name, such as `FedEx`.
	//
	// Unique (case-insensitive) among your account's carriers and the system carriers.
	Name string `json:"name" validate:"required"`
	// What kind of carrier this is.
	//
	// `fedex`, `ups`, `usps`: connected through Shippo for live rates and labels. `ltl`: less-than-truckload freight. `will_call`: the customer picks up. `local_delivery`: your own vehicles. `other`: any other carrier you manage yourself.
	Type constants.CarrierType `json:"type" validate:"required"`
	// Standard Carrier Alpha Code, such as `FDEG`.
	//
	// The 2–4 letter code that identifies the carrier on EDI documents and bills of lading. `null` when not set.
	SCAC *string `json:"scac"`
	// Your account number with the carrier.
	//
	// Required for `ups` and `usps`, which connect using it. FedEx connects through OAuth instead.
	AccountNumber *string `json:"account_number"`
	// Whether customers can see and pick this carrier at checkout in the customer portal.
	PortalVisibility constants.CustomerPortalVisibility `json:"portal_visibility" validate:"required"`
	// Whether the carrier can be newly assigned.
	//
	// `active` carriers can be set on orders and customer defaults. `archived` carriers stay on the records that use them but can't be newly set.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// Service level selected by default when this carrier is chosen.
	//
	// `null` when the carrier has no default.
	DefaultServiceLevel *ServiceLevel `json:"default_service_level" expandable:"true"`
	// Provenance of this carrier.
	//
	// System-owned carriers are platform-provided defaults shared across all accounts and cannot be updated or deleted; account-owned carriers are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Service levels this carrier offers.
	//
	// Returns the first 10. When there are more, `page_info.next_page_url` fetches the rest.
	ServiceLevels *List[ServiceLevel] `json:"service_levels" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleCarrierAccountNumber = "603145678"
var sampleCarrierSCAC = "FDEG"

var SampleCarrier = &Carrier{
	ID:                  SampleCarrierID,
	Object:              constants.ObjectTypeCarrier,
	Name:                SampleCarrierName,
	Type:                constants.CarrierTypeFedEx,
	SCAC:                &sampleCarrierSCAC,
	AccountNumber:       &sampleCarrierAccountNumber,
	PortalVisibility:    constants.CustomerPortalVisibilityVisible,
	Status:              constants.ArchiveStatusActive,
	DefaultServiceLevel: SampleServiceLevel,
	Owner:               SampleOwnerAccount,
	ServiceLevels:       NewList([]ServiceLevel{*SampleServiceLevel}, PageInfo{}),
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Carrier) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCarrier)
}

// Response from initiating carrier OAuth.
type OAuthResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=oauth_response"`
	// URL to send the user to so they can authorize their carrier account.
	//
	// Once the user finishes authorizing, the carrier returns them to the `redirect_uri` supplied when the flow was initiated.
	OAuthURL string `json:"oauth_url" validate:"required"`
}

var SampleOAuthResponse = &OAuthResponse{
	Object:   constants.ObjectTypeOAuthResponse,
	OAuthURL: "https://oauth.fedex.com/authorize?client_id=abc123&redirect_uri=https://www.openmrp.ai/carriers/oauth/callback",
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
	// - `connected`: your own carrier account is authorized and ready for live rating and label purchase.
	// - `authorization_pending`: a carrier account exists but is still Shippo's shared default account, so authorization of your own carrier account has not been completed.
	// - `disconnected`: the carrier has no carrier account to authorize, or the carrier account could not be reached. Sandbox accounts always report this status.
	Status constants.CarrierConnectionStatus `json:"status" validate:"required"`
}

var SampleOAuthStatusResponse = &OAuthStatusResponse{
	Object: constants.ObjectTypeOAuthStatusResponse,
	Status: constants.CarrierConnectionStatusConnected,
}

func (*OAuthStatusResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOAuthStatusResponse)
}
