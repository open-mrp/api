package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePortalRegistrationSessionID = "porgse_017513382536fd23a343e958ef"

// PortalRegistrationSessionData is the scratch form data accumulated across a buyer's registration steps, echoed back so a resumed session restores the form.
type PortalRegistrationSessionData struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_registration_session_data"`
	// The customer's name.
	CustomerName string `json:"customer_name"`
	// An existing customer number to link.
	CustomerNumber string `json:"customer_number"`
	// The chosen customer group ID.
	CustomerGroupID string `json:"customer_group_id"`
	// The chosen payment term ID.
	PaymentTermID string `json:"payment_term_id"`
	// The chosen shipping term ID.
	ShippingTermID string `json:"shipping_term_id"`
	// Contact phone number.
	Phone string `json:"phone"`
	// Billing address name.
	AddressName string `json:"address_name"`
	// Billing address street line 1.
	AddressStreet1 string `json:"address_street_1"`
	// Billing address street line 2.
	AddressStreet2 string `json:"address_street_2"`
	// Billing address city / locality.
	AddressLocality string `json:"address_locality"`
	// Billing address state.
	AddressState string `json:"address_state"`
	// Billing address postal code.
	AddressPostalCode string `json:"address_postal_code"`
	// Billing address two-letter country code.
	AddressCountry string `json:"address_country"`
}

// PortalRegistrationSession is a buyer's session-based registration into a seller's customer portal. The buyer creates or resumes a session, advances it step by step, and completes it — so a half-finished registration can be resumed rather than leaving the buyer stuck.
type PortalRegistrationSession struct {
	// Portal registration session ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_registration_session"`
	// The seller account this registration is for.
	SellerAccountID string `json:"seller_account_id" validate:"required"`
	// The seller's portal slug.
	SellerSlug string `json:"seller_slug" validate:"required"`
	// The user who registered.
	UserID string `json:"user_id" validate:"required"`
	// Whether the buyer is linking an existing customer record vs. creating a new one.
	IsExistingCustomer *bool `json:"is_existing_customer"`
	// The current registration step.
	Step constants.PortalRegistrationStep `json:"step" validate:"required"`
	// Derived lifecycle status, so customer service can spot registrations that stalled.
	Status constants.PortalRegistrationStatus `json:"status" validate:"required"`
	// The customer account created/linked on completion.
	CustomerID *string `json:"customer_id"`
	// Scratch form data accumulated across steps.
	SessionData *PortalRegistrationSessionData `json:"session_data"`
	// When the registration completed, or null while in progress.
	CompletedAt *time.Time `json:"completed_at"`
	// When the session was abandoned, or null.
	AbandonedAt *time.Time `json:"abandoned_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePortalRegistrationSession = &PortalRegistrationSession{
	ID:              SamplePortalRegistrationSessionID,
	Object:          constants.ObjectTypePortalRegistrationSession,
	SellerAccountID: SampleAccountID,
	SellerSlug:      SampleAccountPortalSlug,
	UserID:          SampleUserID,
	Step:            constants.PortalRegistrationStepCustomerDetails,
	Status:          constants.PortalRegistrationStatusInProgress,
	SessionData: &PortalRegistrationSessionData{
		Object:       constants.ObjectTypePortalRegistrationSessionData,
		CustomerName: "Acme Corp",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*PortalRegistrationSession) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePortalRegistrationSession)
}
