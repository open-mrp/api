package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRegistrationSessionID = "rgfw_01gf7a8200eaj8fke1xvw4h50x"
const SampleCheckoutSessionID = "cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG"
const SampleAddressID = "ad_01gf7a8200eaj8fke1xvw4h50x"
const SampleAddressLine1 = "123 Main Street"
const SampleAddressLine2 = "Suite 100"
const SampleAddressCity = "San Francisco"
const SampleAddressState = "CA"
const SampleAddressPostalCode = "94105"
const SampleAddressCountry = "US"

// User data within a registration session.
type RegistrationSessionUser struct {
	// User ID, null until user is created.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// Email address.
	Email string `json:"email" validate:"required"`
	// Timestamp when email was verified, null if pending.
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	// Display name.
	Name *string `json:"name"`
}

// Account data within a registration session.
type RegistrationSessionAccount struct {
	// Account ID, null until account is created.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Billing address.
	BillingAddress RegistrationSessionAddress `json:"billing_address" validate:"required"`
}

// Address within a registration session.
type RegistrationSessionAddress struct {
	// Address ID, null until address is created.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=address"`
	// Street address line 1.
	Line1 *string `json:"line1"`
	// Street address line 2 (apartment, suite, etc.).
	Line2 *string `json:"line2"`
	// City name.
	City *string `json:"city"`
	// State or province.
	State *string `json:"state"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code"`
	// Two-letter country code.
	Country *string `json:"country"`
}

// Registration session.
type RegistrationSession struct {
	// Session ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_session"`
	// Pricing plan code.
	PlanCode string `json:"plan_code" validate:"required"`
	// Current registration step.
	Step constants.RegistrationStep `json:"step" validate:"required"`
	// Stripe customer ID.
	StripeCustomerID *string `json:"stripe_customer_id"`
	// Stripe checkout session ID.
	StripeCheckoutSessionID *string `json:"stripe_checkout_session_id"`
	// Whether payment has been completed.
	PaymentCompleted bool `json:"payment_completed"`
	// Account being registered.
	Account *RegistrationSessionAccount `json:"account"`
	// User being registered.
	User RegistrationSessionUser `json:"user" validate:"required"`
	// Timestamp when registration was completed. Null if still in progress.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when this session was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when this session was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Result of creating a registration session.
type CreateSessionResponse struct {
	// Session ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_session"`
}

// Result of completing a registration.
type CompleteRegistrationResponse struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
}

// Result of creating a user for a registration session.
type CreateUserResponse struct {
	// User ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
}

// Result of setting up billing for a registration.
type SetupBillingResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=setup_billing_response"`
	// Stripe customer ID.
	StripeCustomerID string `json:"stripe_customer_id" validate:"required"`
	// Stripe Setup Intent client secret for Stripe.js payment collection.
	ClientSecret string `json:"client_secret" validate:"required"` // #nosec G117 -- Stripe client_secret passed to frontend, not a hardcoded secret
	// Stripe publishable key for Stripe.js initialization.
	PublishableKey string `json:"publishable_key" validate:"required"`
}

// Result of confirming payment for a registration.
type ConfirmPaymentResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=confirm_payment_response"`
	// Setup Intent status (e.g., "succeeded").
	Status string `json:"status" validate:"required"`
	// Payment method ID attached by the Setup Intent.
	PaymentMethodID *string `json:"payment_method_id"`
}

var SampleRegistrationSessionAddress = &RegistrationSessionAddress{
	ID:         new(SampleAddressID),
	Object:     constants.ObjectTypeAddress,
	Line1:      new(SampleAddressLine1),
	Line2:      new(SampleAddressLine2),
	City:       new(SampleAddressCity),
	State:      new(SampleAddressState),
	PostalCode: new(SampleAddressPostalCode),
	Country:    new(SampleAddressCountry),
}

var SampleRegistrationSessionUser = &RegistrationSessionUser{
	ID:              new(SampleUserID),
	Object:          constants.ObjectTypeUser,
	Email:           SampleUserEmail,
	EmailVerifiedAt: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	Name:            new(SampleUserName),
}

var SampleRegistrationSessionAccount = &RegistrationSessionAccount{
	ID:             new(SampleAccountID),
	Object:         constants.ObjectTypeAccount,
	Name:           SampleAccountName,
	BillingAddress: *SampleRegistrationSessionAddress,
}

var SampleRegistrationSession = &RegistrationSession{
	ID:                      SampleRegistrationSessionID,
	Object:                  constants.ObjectTypeRegistrationSession,
	PlanCode:                string(constants.PlanCodeStarter),
	Step:                    constants.RegistrationStepVerification,
	StripeCustomerID:        new(SampleStripeCustomerID),
	StripeCheckoutSessionID: new(SampleCheckoutSessionID),
	PaymentCompleted:        false,
	Account:                 SampleRegistrationSessionAccount,
	User:                    *SampleRegistrationSessionUser,
	CompletedAt:             nil,
	CreatedAt:               timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:               timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

var SampleCreateSessionResponse = &CreateSessionResponse{
	ID:     SampleRegistrationSessionID,
	Object: constants.ObjectTypeRegistrationSession,
}

var SampleCompleteRegistrationResponse = &CompleteRegistrationResponse{
	ID:     SampleAccountID,
	Object: constants.ObjectTypeAccount,
}

var SampleCreateUserResponse = &CreateUserResponse{
	ID:     SampleUserID,
	Object: constants.ObjectTypeUser,
}

var SampleSetupBillingResponse = &SetupBillingResponse{
	Object:           constants.ObjectTypeSetupBillingResponse,
	StripeCustomerID: SampleStripeCustomerID,
	ClientSecret:     "seti_1234_secret_5678",
	PublishableKey:   "pk_test_example",
}

var SampleConfirmPaymentResponse = &ConfirmPaymentResponse{
	Object: constants.ObjectTypeConfirmPaymentResponse,
	Status: "succeeded",
}

func (*RegistrationSessionUser) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSessionUser)
}

func (*RegistrationSessionAccount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSessionAccount)
}

func (*RegistrationSessionAddress) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSessionAddress)
}

func (*RegistrationSession) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSession)
}

func (*CreateSessionResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreateSessionResponse)
}

func (*CompleteRegistrationResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCompleteRegistrationResponse)
}

func (*CreateUserResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreateUserResponse)
}

func (*SetupBillingResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSetupBillingResponse)
}

func (*ConfirmPaymentResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConfirmPaymentResponse)
}
