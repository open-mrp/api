package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleRegistrationSessionID = "rs_01gf7a8200eaj8fke1xvw4h50x"
const SampleCheckoutSessionID = "cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG"
const SampleAddressID = "ad_01gf7a8200eaj8fke1xvw4h50x"
const SampleAddressLine1 = "123 Main Street"
const SampleAddressLine2 = "Suite 100"
const SampleAddressCity = "San Francisco"
const SampleAddressState = "CA"
const SampleAddressPostalCode = "94105"
const SampleAddressCountry = "US"

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
	ID:            new(SampleUserID),
	Object:        constants.ObjectTypeUser,
	Email:         SampleUserEmail,
	EmailVerified: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	Name:          new(SampleUserName),
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

// RegistrationSessionUser represents user data within a registration session.
type RegistrationSessionUser struct {
	// The user ID, null until user is created.
	ID *string `json:"id"`
	// Object type identifier for registration session user.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// Email address provided during registration.
	Email string `json:"email" validate:"required"`
	// Timestamp when email was verified, null if pending.
	EmailVerified *time.Time `json:"email_verified"`
	// Display name provided during registration.
	Name *string `json:"name"`
}

func (*RegistrationSessionUser) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSessionUser)
}

// RegistrationSessionAccount represents account data within a registration session.
type RegistrationSessionAccount struct {
	// The account ID, null until account is created.
	ID *string `json:"id"`
	// Object type identifier for registration session account.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The account's display name.
	Name string `json:"name" validate:"required"`
	// The account's billing address.
	BillingAddress RegistrationSessionAddress `json:"billing_address" validate:"required"`
}

func (*RegistrationSessionAccount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSessionAccount)
}

// RegistrationSessionAddress represents an address within a registration session.
type RegistrationSessionAddress struct {
	// The address ID, null until address is created.
	ID *string `json:"id"`
	// Object type identifier for registration session address.
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

func (*RegistrationSessionAddress) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSessionAddress)
}

// RegistrationSession represents a registration session.
type RegistrationSession struct {
	// The unique identifier for this registration session.
	ID string `json:"id" validate:"required"`
	// The type of this object.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_session"`
	// The pricing plan code selected during registration.
	PlanCode string `json:"plan_code" validate:"required"`
	// The current step in the registration flow.
	Step constants.RegistrationStep `json:"step" validate:"required"`
	// The Stripe customer ID, if one has been created.
	StripeCustomerID *string `json:"stripe_customer_id"`
	// The Stripe checkout session ID, if checkout was initiated.
	StripeCheckoutSessionID *string `json:"stripe_checkout_session_id"`
	// Whether payment has been completed for this registration.
	PaymentCompleted bool `json:"payment_completed"`
	// The account being registered.
	Account *RegistrationSessionAccount `json:"account"`
	// The user being registered.
	User RegistrationSessionUser `json:"user" validate:"required"`
	// When the registration was completed, null if still in progress.
	CompletedAt *time.Time `json:"completed_at"`
	// When this registration session was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this registration session was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

func (*RegistrationSession) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRegistrationSession)
}

var SampleCreateSessionResponse = &CreateSessionResponse{
	ID:     SampleRegistrationSessionID,
	Object: constants.ObjectTypeRegistrationSession,
}

var SampleCompleteRegistrationResponse = &CompleteRegistrationResponse{
	ID:     SampleAccountID,
	Object: constants.ObjectTypeAccount,
}

// CreateSessionResponse is the response from creating a registration session.
type CreateSessionResponse struct {
	// The unique identifier of the created registration session.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=registration_session"`
}

func (*CreateSessionResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreateSessionResponse)
}

// CompleteRegistrationResponse is the response from completing a registration.
type CompleteRegistrationResponse struct {
	// The ID of the created account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
}

func (*CompleteRegistrationResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCompleteRegistrationResponse)
}

var SampleCreateUserResponse = &CreateUserResponse{
	ID:     SampleUserID,
	Object: constants.ObjectTypeUser,
}

// CreateUserResponse is the response from creating a user for a registration session.
type CreateUserResponse struct {
	// The ID of the created user.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
}

func (*CreateUserResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreateUserResponse)
}

var SampleCreateCheckoutResponse = &CreateCheckoutResponse{
	ClientSecret:     SampleClientSecret,
	CheckoutID:       SampleCheckoutSessionID,
	StripeCustomerID: SampleStripeCustomerID,
	PublishableKey:   SamplePublishableKey,
}

// CreateCheckoutResponse is the response from creating a checkout session for a registration.
type CreateCheckoutResponse struct {
	// The Stripe checkout session client secret for the embedded checkout UI.
	ClientSecret string `json:"client_secret" validate:"required"` // #nosec G117 - Stripe checkout client secret (ephemeral, not a credential)
	// The Stripe checkout session ID.
	CheckoutID string `json:"checkout_id" validate:"required"`
	// The Stripe customer ID created for this registration.
	StripeCustomerID string `json:"stripe_customer_id" validate:"required"`
	// The Stripe publishable key for the frontend.
	PublishableKey string `json:"publishable_key" validate:"required"`
}

func (*CreateCheckoutResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreateCheckoutResponse)
}

var SampleConfirmPaymentResponse = &ConfirmPaymentResponse{
	Status:           "complete",
	SubscriptionID:   new(SampleSubscriptionID),
	StripeCustomerID: new(SampleStripeCustomerID),
}

// ConfirmPaymentResponse is the response from confirming a registration payment.
type ConfirmPaymentResponse struct {
	// The Stripe checkout session status (e.g., "complete", "open", "expired").
	Status string `json:"status" validate:"required"`
	// The Stripe subscription ID, present when payment is complete.
	SubscriptionID *string `json:"subscription_id"`
	// The Stripe customer ID associated with the checkout.
	StripeCustomerID *string `json:"stripe_customer_id"`
}

func (*ConfirmPaymentResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConfirmPaymentResponse)
}
