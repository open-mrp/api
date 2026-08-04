package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRegistrationSessionID = "rgfw_6xab8u2fun46"
const SampleCheckoutSessionID = "cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG"
const SampleAddressID = "ad_npqa5y43q26z"
const SampleAddressLine1 = "123 Main Street"
const SampleAddressLine2 = "Suite 100"
const SampleAddressCity = "San Francisco"
const SampleAddressState = "CA"
const SampleAddressPostalCode = "94105"
const SampleAddressCountry = "US"

// User data within a registration session.
type RegistrationSessionUser struct {
	// ID of the user record.
	//
	// Populated once the user is created during the `user_details` step.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// Email address.
	Email string `json:"email" validate:"required"`
	// The user's display name.
	//
	// Provided by the registrant during the `user_details` step.
	Name *string `json:"name"`
	// When the user's email address was verified.
	//
	// Set once the registrant follows the link in the verification email. It mirrors the session's `updated_at` timestamp rather than recording the moment of verification, so it moves forward as the rest of the registration is filled in.
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
}

// Account data within a registration session.
type RegistrationSessionAccount struct {
	// ID of the account record.
	//
	// Populated only after the registration completes and the account is provisioned.
	ID *string `json:"id"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name of the account being created.
	Name string `json:"name" validate:"required"`
	// Address the account will be billed at.
	//
	// Also becomes the new account's business address when the registration completes.
	BillingAddress RegistrationSessionAddress `json:"billing_address" validate:"required"`
}

// Address within a registration session.
type RegistrationSessionAddress struct {
	// ID of the address record.
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

// An in-progress self-serve registration.
//
// A session tracks a new customer's progress through email verification, user and account setup, payment, and final account provisioning.
type RegistrationSession struct {
	// Session ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_session"`
	// Code of the pricing plan selected for this registration.
	PlanCode string `json:"plan_code" validate:"required"`
	// Current step in the registration flow.
	//
	// Steps advance in this order:
	//
	// - `verification`: the user is verifying their email address.
	// - `user_details`: the user is providing their personal details (name, etc.).
	// - `account_details`: the user is providing their account/company details.
	// - `review`: the user is reviewing their registration details before payment.
	// - `payment`: the user is providing their payment details.
	// - `completed`: registration has finished and the account is active.
	Step constants.RegistrationStep `json:"step" validate:"required"`
	// ID of the Stripe customer created for this registration.
	//
	// Populated when Setup Registration Billing runs; absent for free plans, which never set up billing.
	StripeCustomerID *string `json:"stripe_customer_id"`
	// ID of the Stripe Setup Intent created to collect the payment method.
	//
	// Despite the field name, this holds the Setup Intent ID created by Setup Registration Billing, and Confirm Registration Payment only accepts a `setup_intent_id` matching it.
	StripeCheckoutSessionID *string `json:"stripe_checkout_session_id"`
	// Whether payment has been completed for this registration.
	//
	// Set to `true` once Confirm Registration Payment verifies the Setup Intent. Free plans never collect payment, so this stays `false` and the registration can still be completed.
	PaymentCompleted bool `json:"payment_completed"`
	// Account being registered.
	//
	// Populated once account details are entered during the `account_details` step.
	Account *RegistrationSessionAccount `json:"account"`
	// User being registered.
	User RegistrationSessionUser `json:"user" validate:"required"`
	// Timestamp when registration was completed.
	CompletedAt *time.Time `json:"completed_at"`
	// Timestamp when this session was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Timestamp when this session was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Result of creating a registration session.
type CreateSessionResponse struct {
	// ID of the registration session.
	//
	// If an active session already existed for the email, this is the existing session's ID rather than a new one.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_session"`
}

// Result of completing a registration.
type CompleteRegistrationResponse struct {
	// ID of the newly created account.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
}

// Result of creating a user for a registration session.
type CreateUserResponse struct {
	// ID of the user associated with the session.
	//
	// Repeating the call on a session that already has a user returns that same user rather than creating another.
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
	ClientSecret string `json:"client_secret" validate:"required" sensitive:"true"` // #nosec G117 -- Stripe client_secret passed to frontend, not a hardcoded secret
	// Stripe publishable key for Stripe.js initialization.
	PublishableKey string `json:"publishable_key" validate:"required"`
}

// Result of confirming payment for a registration.
type ConfirmPaymentResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=confirm_payment_response"`
	// Status of the Stripe Setup Intent.
	//
	// Always `succeeded` on a successful response; any other Setup Intent status results in a validation error instead.
	Status string `json:"status" validate:"required"`
	// Payment method ID attached by the Setup Intent.
	//
	// Returned only the first time payment is confirmed; a repeat confirmation of an already-completed session succeeds but omits it.
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
	Name:            new(SampleUserName),
	EmailVerifiedAt: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
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
	Object:          constants.ObjectTypeConfirmPaymentResponse,
	Status:          "succeeded",
	PaymentMethodID: new("pm_1QXmZ2AbCdEfGhIjKlMnOpQr"),
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
