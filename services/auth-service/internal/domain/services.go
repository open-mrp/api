package domain

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/apikey"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/ptrutil"
)

type LoginResult struct {
	User         *types.User
	RefreshToken string // #nosec G117 - Struct field, not a hardcoded credential
	AccessToken  string // #nosec G117 - Struct field, not a hardcoded credential
}

type RefreshTokenResult struct {
	AccessToken string // #nosec G117 - Struct field, not a hardcoded credential
}

type RegisterInput struct {
	Name        string
	Email       string
	Password    string  // #nosec G117 - Struct field, not a hardcoded credential
	AccountSlug *string // Portal context for the "already registered" magic login link.
	// PortalBaseURL is the base URL of the account's verified custom portal domain, resolved server-side by the gateway. When set, email links use it instead of the slug-prefixed dashboard URL.
	PortalBaseURL *string
}

type AuthSvc interface {
	// ValidateCredential validates an auth token and returns the resulting identity.
	//
	// Behavior:
	//   - If authToken is empty, returns an unauthenticated identity.
	//   - If authToken is an API key credential, validates it as an API key.
	//   - Otherwise, validates it as a user credential.
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError)
}

type TokenSvc interface {
	// RefreshToken exchanges a valid refresh token for a new short-lived access token.
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResult, *apierror.APIError)

	// RevokeRefreshToken invalidates a refresh token so it can no longer be used to mint access tokens.
	RevokeRefreshToken(ctx context.Context, refreshToken string) *apierror.APIError
}

type UserSvc interface {
	// Login authenticates a user and returns a token pair (access + refresh) plus the user profile.
	Login(ctx context.Context, identifier, password string) (*LoginResult, *apierror.APIError)

	// Register creates a new user and returns a token pair so the user is immediately logged in.
	Register(ctx context.Context, input RegisterInput) (*LoginResult, *apierror.APIError)

	// MagicLogin exchanges a magic-login token for a token pair, logging the user in without a password.
	MagicLogin(ctx context.Context, token string) (*LoginResult, *apierror.APIError)
}

type PasswordSvc interface {
	// UpdatePassword updates a user's password.
	//
	// Side effects:
	//   - Revokes existing refresh tokens for the user.
	UpdatePassword(ctx context.Context, oldPassword, newPassword string) *apierror.APIError

	// ResetPassword completes the password reset flow and returns a token pair plus the user profile.
	//
	// Side effects:
	//   - Revokes existing refresh tokens for the user.
	ResetPassword(ctx context.Context, token, newPassword string) (*LoginResult, *apierror.APIError)

	// RequestPasswordReset initiates a password reset flow for the identifier.
	RequestPasswordReset(ctx context.Context, identifier string, accountSlug, portalBaseURL *string) *apierror.APIError
}

type CreateRegistrationSessionInput struct {
	Email    string
	PlanCode string
}

type CreateRegistrationSessionResult struct {
	SessionID string
}

type CreateUserForRegistrationInput struct {
	SessionID string
	Name      string
	Password  string // #nosec G117 - Struct field, not a hardcoded credential
}

type CreateUserForRegistrationOutput struct {
	UserID       string
	AccessToken  string // #nosec G117 - Struct field, not a hardcoded credential
	RefreshToken string // #nosec G117 - Struct field, not a hardcoded credential
}

type UpdateRegistrationSessionData struct {
	UserName                 *string
	AccountName              *string
	BillingAddressLine1      *string
	BillingAddressLine2      *string
	BillingAddressCity       *string
	BillingAddressState      *string
	BillingAddressPostalCode *string
	BillingAddressCountry    *string
}

// MergeInto applies non-nil fields from the update into the target, leaving fields that were not provided in the PATCH request unchanged.
func (u *UpdateRegistrationSessionData) MergeInto(target *RegistrationSessionData) {
	ptrutil.ApplyIfSet(&target.UserName, u.UserName)
	ptrutil.ApplyIfSet(&target.AccountName, u.AccountName)
	ptrutil.ApplyIfSet(&target.BillingAddressLine1, u.BillingAddressLine1)
	ptrutil.ApplyIfSet(&target.BillingAddressLine2, u.BillingAddressLine2)
	ptrutil.ApplyIfSet(&target.BillingAddressCity, u.BillingAddressCity)
	ptrutil.ApplyIfSet(&target.BillingAddressState, u.BillingAddressState)
	ptrutil.ApplyIfSet(&target.BillingAddressPostalCode, u.BillingAddressPostalCode)
	ptrutil.ApplyIfSet(&target.BillingAddressCountry, u.BillingAddressCountry)
}

type UpdateRegistrationSessionInput struct {
	SessionID   string
	Step        *constants.RegistrationStep
	SessionData *UpdateRegistrationSessionData
}

type SetupBillingInput struct {
	SessionID string
}

type SetupBillingOutput struct {
	StripeCustomerID string
	ClientSecret     string // #nosec G117 -- Stripe ephemeral client secret
	PublishableKey   string
}

type ConfirmPaymentInput struct {
	SessionID     string
	SetupIntentID string
}

type ConfirmPaymentOutput struct {
	Status          string
	PaymentMethodID *string
}

type ListRegistrationSessionsInput struct {
	Cursor *string
	Limit  int32
}

type ListRegistrationSessionsResult struct {
	Sessions []*RegistrationSession
	PageInfo pagination.PageInfo
}

type RegistrationSessionSvc interface {
	// CreateSession creates a new registration session or returns an existing active (uncompleted) session for the given email.
	//
	// Side effects:
	//   - Sends a verification email to the user.
	CreateSession(ctx context.Context, input CreateRegistrationSessionInput) (*CreateRegistrationSessionResult, *apierror.APIError)

	// ResendVerificationEmail regenerates the verification token for an existing registration session and resends the verification email.
	//
	// Side effects:
	//   - Rotates the verification token.
	//   - Sends a new verification email to the user.
	ResendVerificationEmail(ctx context.Context, sessionID string) *apierror.APIError

	// VerifyToken verifies the email token from the registration verification link. Marks the session's email as verified and advances the step to user_details. Idempotent: repeated calls return the same session.
	VerifyToken(ctx context.Context, token string) (*RegistrationSession, *apierror.APIError)

	// GetSession returns the current state of a registration session by its type ID. Returns a not-found error if the session does not exist.
	GetSession(ctx context.Context, sessionID string) (*RegistrationSession, *apierror.APIError)

	// CreateUserForSession creates or resolves a user for a registration session and returns the user ID with auth tokens.
	//
	// Side effects:
	//   - Creates a new user if one does not already exist for the session email.
	//   - Associates the user with the registration session.
	//   - Advances the session step to account_details.
	CreateUserForSession(ctx context.Context, input CreateUserForRegistrationInput) (*CreateUserForRegistrationOutput, *apierror.APIError)

	// UpdateSession updates an in-progress registration session's step, form data, and/or Stripe-related fields. Returns the updated session.
	//
	// Authorization:
	//   - Requires a user identity in context.
	UpdateSession(ctx context.Context, input UpdateRegistrationSessionInput) (*RegistrationSession, *apierror.APIError)

	// ListSessions returns a paginated list of open (uncompleted) registration sessions for the authenticated user.
	//
	// Authorization:
	//   - Requires a user identity in context.
	ListSessions(ctx context.Context, input ListRegistrationSessionsInput) (*ListRegistrationSessionsResult, *apierror.APIError)

	// SetupBilling creates a Stripe customer and Setup Intent for a registration session. Uses recovery points for crash safety.
	//
	// Authorization:
	//   - Requires a user identity in context.
	SetupBilling(ctx context.Context, input SetupBillingInput) (*SetupBillingOutput, *apierror.APIError)

	// ConfirmPayment verifies that a Setup Intent succeeded and marks the registration session's payment as completed.
	//
	// Authorization:
	//   - Requires a user identity in context matching the session's user.
	ConfirmPayment(ctx context.Context, input ConfirmPaymentInput) (*ConfirmPaymentOutput, *apierror.APIError)

	// CompleteRegistration finalizes a registration session by calling core-service to create the production account, sandbox, roles, and permissions, then marks the session as completed.
	//
	// Authorization:
	//   - Requires a user identity in context matching the session's user.
	CompleteRegistration(ctx context.Context, sessionID string) (*CompleteRegistrationOutput, *apierror.APIError)

	// GetIncompleteByUserID returns the most recent incomplete registration session for the given user, or (nil, nil) if none exists.
	GetIncompleteByUserID(ctx context.Context, userID string) (*RegistrationSession, *apierror.APIError)
}

// CompleteRegistrationOutput holds the IDs of the newly created accounts.
type CompleteRegistrationOutput struct {
	AccountID string
	SandboxID string
}

// CompleteAccountRegistrationInput carries the data sent to core-service to create the account and sandbox.
type CompleteAccountRegistrationInput struct {
	UserID           string
	PlanCode         string
	StripeCustomerID string
	AccountName      string
	UserName         string
	UserEmail        string
	BusinessAddress  *RegistrationAddress
}

// RegistrationAddress is a structured postal address collected during registration.
type RegistrationAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

type CreateAPIKeyInput struct {
	RoleID    string
	Name      string
	ExpiresAt *time.Time
}

type CreateAPIKeyResult struct {
	APIKeySecret string
	APIKey       *apikey.APIKey
}

type RotateAPIKeyInput struct {
	APIKeyID  string
	ExpiresAt *time.Time
	// RevokeAt schedules when the old key is revoked. Nil or past/now means immediate revocation; a future instant keeps the old key valid until then.
	RevokeAt *time.Time
}

type RevokeAPIKeyInput struct {
	APIKeyID string
}

type ListAPIKeysResult struct {
	APIKeys  []*apikey.APIKey
	PageInfo pagination.PageInfo
}

type GetOrCreateDocAPIKeyResult struct {
	APIKeySecret string
	APIKey       *apikey.APIKey
}

type DocAPIKeySvc interface {
	// GetOrCreateDocAPIKey returns a documentation API key for the caller's target account.
	//
	// Authorization:
	//   - Requires an internal identity with a target account in context.
	//
	// Behavior:
	//   - Reuses an existing valid key.
	//   - Rotates and replaces an expired key.
	//
	// Side effects:
	//   - May rotate (revoke and replace) an existing doc API key.
	GetOrCreateDocAPIKey(ctx context.Context) (*GetOrCreateDocAPIKeyResult, *apierror.APIError)
}

type APIKeySvc interface {
	// GetAPIKey returns a single API key's metadata by its type ID.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	//   - The key must belong to the caller's target account.
	GetAPIKey(ctx context.Context, apiKeyID string, includes []string) (*apikey.APIKey, *apierror.APIError)

	// CreateAPIKey creates a new API key for the caller's target account.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	CreateAPIKey(ctx context.Context, input CreateAPIKeyInput) (*CreateAPIKeyResult, *apierror.APIError)

	// RotateAPIKey rotates (revokes and replaces) an API key.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	//
	// Side effects:
	//   - Revokes the prior API key.
	RotateAPIKey(ctx context.Context, input RotateAPIKeyInput) (*CreateAPIKeyResult, *apierror.APIError)

	// RevokeAPIKey revokes an API key without creating a replacement.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	RevokeAPIKey(ctx context.Context, input RevokeAPIKeyInput) *apierror.APIError

	// ListAPIKeys returns a paginated list of API keys for the caller's target account.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	//
	// Pagination:
	//   - If cursor is non-nil, results begin after the provided cursor.
	//   - limit controls the maximum number of results returned.
	ListAPIKeys(ctx context.Context, cursor *string, limit int32, query *string, statuses []constants.APIKeyStatus, includes []string) (*ListAPIKeysResult, *apierror.APIError)

	BatchGetAPIKeysByIDs(ctx context.Context, ids []string) ([]*apikey.APIKey, *apierror.APIError)
}
