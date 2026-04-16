package domain

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
)

type APIKeyListRepoInput struct {
	OwnerAccountID string
	Cursor         *string
	Limit          int32
	Query          *string
	Statuses       []constants.APIKeyStatus
	Includes       []string
}

type APIKeyListRepoResult struct {
	APIKeys  []*apikey.APIKey
	PageInfo pagination.PageInfo
}

type APIKeyRepo interface {
	Find(ctx context.Context, apiKeyID string) (*apikey.APIKey, *apierror.APIError)
	FindByDatabaseID(ctx context.Context, id int64, includes []string) (*apikey.APIKey, *apierror.APIError)
	FindByTypeID(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError)
	Touch(ctx context.Context, apiKeyID int64) *apierror.APIError
	Create(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError)
	Revoke(ctx context.Context, typeID string) *apierror.APIError
	List(ctx context.Context, input APIKeyListRepoInput) (*APIKeyListRepoResult, *apierror.APIError)
}

type RefreshTokenRepo interface {
	Find(ctx context.Context, token string) (*RefreshToken, *apierror.APIError)
	Create(ctx context.Context, userID string, token string, expiresInDays int) (*RefreshToken, *apierror.APIError)
	Revoke(ctx context.Context, token string) *apierror.APIError
	RevokeAll(ctx context.Context, userID string) *apierror.APIError
}

type UserRepo interface {
	Find(ctx context.Context, identifier string) (*types.User, *apierror.APIError)
	Create(ctx context.Context, userID, email, name, hashedPassword string) (*types.User, *apierror.APIError)
	UpdatePassword(ctx context.Context, userID string, hashedPassword string) *apierror.APIError
}

type DocAPIKeyRepo interface {
	FindBySandboxAccountID(ctx context.Context, sandboxAccountID string) (*apikey.DocAPIKey, *apierror.APIError)
	FindByAPIKeyID(ctx context.Context, apiKeyID string) (*apikey.DocAPIKey, *apierror.APIError)
	Create(ctx context.Context, docAPIKey *apikey.DocAPIKey) (int64, *apierror.APIError)
	Update(ctx context.Context, docAPIKey *apikey.DocAPIKey) *apierror.APIError
	Delete(ctx context.Context, id int64) *apierror.APIError
}

type RegistrationSessionRepo interface {
	// GetByEmail returns the most recent active (uncompleted) registration session
	// for the given email. Returns a not-found error if none exists.
	GetByEmail(ctx context.Context, email string) (*RegistrationSession, *apierror.APIError)

	// GetByTypeID returns the registration session with the given type ID.
	// Returns a not-found error if none exists.
	GetByTypeID(ctx context.Context, typeID string) (*RegistrationSession, *apierror.APIError)

	// GetByToken returns the registration session matching the verification token.
	// Returns a not-found error if no session has the given token.
	GetByToken(ctx context.Context, token string) (*RegistrationSession, *apierror.APIError)

	// GetByID returns the registration session with the given database ID.
	// Returns a not-found error if none exists.
	GetByID(ctx context.Context, id int64) (*RegistrationSession, *apierror.APIError)

	// GetIncompleteByUserID returns the most recent incomplete registration
	// session for the given user, or (nil, nil) if none exists.
	GetIncompleteByUserID(ctx context.Context, userID string) (*RegistrationSession, *apierror.APIError)

	// Create persists a new registration session and returns the database-assigned ID.
	Create(ctx context.Context, session *RegistrationSession) (int64, *apierror.APIError)

	// UpdatePlanCode changes the plan code on a registration session.
	UpdatePlanCode(ctx context.Context, id int64, planCode string) *apierror.APIError

	// UpdateToken replaces the verification token for the given session.
	UpdateToken(ctx context.Context, id int64, verificationToken string) *apierror.APIError

	// UpdateEmailVerified marks the session's email as verified and sets the
	// is_existing_user flag.
	UpdateEmailVerified(ctx context.Context, id int64, isExistingUser *bool) *apierror.APIError

	// UpdateStep advances the session to the given step and persists the session data.
	UpdateStep(ctx context.Context, id int64, step constants.RegistrationStep, sessionData RegistrationSessionData) *apierror.APIError

	// UpdateUser sets the user ID on the session and persists updated session data.
	UpdateUser(ctx context.Context, id int64, userID string, sessionData RegistrationSessionData) *apierror.APIError

	// UpdateStripeCustomer sets the Stripe customer ID and checkout session ID
	// on the registration session.
	UpdateStripeCustomer(ctx context.Context, id int64, stripeCustomerID *string, stripeCheckoutSessionID *string) *apierror.APIError

	// UpdatePaymentCompleted sets the payment completed flag and Stripe
	// subscription ID on the registration session.
	UpdatePaymentCompleted(ctx context.Context, id int64, paymentCompleted bool, stripeSubscriptionID *string) *apierror.APIError

	// ListByUserID returns open (uncompleted) registration sessions for the
	// given user with cursor-based pagination.
	ListByUserID(ctx context.Context, userID string, cursor *string, limit int32) ([]*RegistrationSession, pagination.PageInfo, *apierror.APIError)

	// UpdateAccountID sets the account ID on the registration session without
	// marking it as completed.
	UpdateAccountID(ctx context.Context, id int64, accountID string) *apierror.APIError

	// Complete marks the registration session as completed and records the
	// account ID.
	Complete(ctx context.Context, id int64, accountID *string) *apierror.APIError
}

type RegistrationQueueRepo interface {
	Create(ctx context.Context, email, name, planCode, registrationSessionID string) *apierror.APIError
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
