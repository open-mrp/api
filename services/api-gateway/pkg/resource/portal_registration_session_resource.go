package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePortalRegistrationSessionID = "porgse_q1hs0mapqh6x"

// The form data a buyer has entered so far in a customer-portal registration.
//
// It is saved on the session as the buyer advances and echoed back on every read, so a resumed registration can restore the form exactly where the buyer left off. The values are used to create or link the customer when the registration is completed.
type PortalRegistrationSessionData struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_registration_session_data"`
	// The name the buyer entered for the customer.
	//
	// Only used when the registration creates a new customer; joining an existing customer keeps that customer's own name.
	CustomerName string `json:"customer_name"`
	// The seller-assigned customer number the buyer is claiming.
	//
	// Only used when the buyer is joining an existing customer, where it must match a customer already on the seller's books. New customers are assigned a number automatically when the registration completes.
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

// A buyer's registration into a seller's customer portal.
//
// The buyer starts a session, advances it step by step, and completes it — so a half-finished registration can be resumed rather than leaving the buyer stuck. Sellers use the same record to see which registrations stalled before completing.
type PortalRegistrationSession struct {
	// Portal registration session ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_registration_session"`
	// The seller account whose portal the buyer is registering into.
	SellerAccountID string `json:"seller_account_id" validate:"required"`
	// The portal slug the registration was started from.
	SellerSlug string `json:"seller_slug" validate:"required"`
	// The buyer this session belongs to.
	//
	// Only this user can retrieve, update, complete, or abandon the session.
	UserID string `json:"user_id" validate:"required"`
	// Whether the buyer is joining a customer the seller already has, rather than creating a new one.
	//
	// When true, completing the registration links the buyer to the seller's existing customer identified by `customer_number`; otherwise it creates a new customer from the rest of the session data.
	IsExistingCustomer *bool `json:"is_existing_customer"`
	// The step the buyer has reached.
	//
	// Steps run `customer_details` → `billing_address` → `contact` → `completed`, and only ever move forward.
	Step constants.PortalRegistrationStep `json:"step" validate:"required"`
	// Where the registration stands, derived from its completion and abandonment timestamps and the seven-day resume window.
	//
	// - `in_progress`: still incomplete and inside the resume window.
	// - `completed`: the buyer finished registering.
	// - `abandoned`: the buyer explicitly gave the session up.
	// - `expired`: still incomplete, but past the resume window, so the buyer can no longer pick it back up.
	Status constants.PortalRegistrationStatus `json:"status" validate:"required"`
	// The customer the registration created or joined.
	CustomerID *string `json:"customer_id"`
	// The form data the buyer has entered so far.
	SessionData *PortalRegistrationSessionData `json:"session_data"`
	// When the buyer completed the registration.
	CompletedAt *time.Time `json:"completed_at"`
	// When the buyer abandoned the session.
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
		Object:            constants.ObjectTypePortalRegistrationSessionData,
		CustomerName:      "Acme Corp",
		CustomerNumber:    "100042",
		CustomerGroupID:   SampleAccountGroupID,
		PaymentTermID:     SamplePaymentTermID,
		ShippingTermID:    SampleShippingTermID,
		Phone:             "555-123-4567",
		AddressName:       "Acme Corp HQ",
		AddressStreet1:    "123 Main St",
		AddressStreet2:    "Suite 400",
		AddressLocality:   "Springfield",
		AddressState:      "IL",
		AddressPostalCode: "62701",
		AddressCountry:    "US",
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*PortalRegistrationSession) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePortalRegistrationSession)
}
