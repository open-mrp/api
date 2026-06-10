package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	pwdutil "github.com/augno/api/services/auth-service/internal/password"
	tokenutil "github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	tracing "github.com/augno/api/shared/tracing"
)

var passwordMedTracer = tracing.GetTracer("auth-service.password_mediator")

type passwordMedImpl struct {
	repos                 domain.RepoFactory
	refreshTokenMed       domain.RefreshTokenMed
	jwtSecret             string
	notificationPublisher domain.NotificationPublisher
	frontendURL           string
}

type PasswordMedConfig struct {
	// Repos (required) is the repository factory for credential persistence.
	Repos domain.RepoFactory

	// RefreshTokenMed (required) revokes refresh tokens on password changes.
	RefreshTokenMed domain.RefreshTokenMed

	// JWTSecret (required) signs and verifies password-flow JWTs.
	JWTSecret string // #nosec G117 - Struct field, not a hardcoded credential

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher

	// FrontendURL (required) is the dashboard base URL used in password emails.
	FrontendURL string
}

func (c *PasswordMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("password mediator: repos is required")
	}
	if c.RefreshTokenMed == nil {
		return fmt.Errorf("password mediator: refresh token mediator is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("password mediator: jwt secret is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("password mediator: notification publisher is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("password mediator: frontend url is required")
	}
	return nil
}

func NewPasswordMed(config *PasswordMedConfig) domain.PasswordMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &passwordMedImpl{
		repos:                 config.Repos,
		refreshTokenMed:       config.RefreshTokenMed,
		jwtSecret:             config.JWTSecret,
		notificationPublisher: config.NotificationPublisher,
		frontendURL:           config.FrontendURL,
	}
}

// Validate validates the identifier/password combination and returns the associated user.
//
//  1. Look up the user by identifier (email or user ID).
//  2. If the user has no stored password hash, silently send a password reset email
//     (when an email is on file) and return the generic invalid-credentials error so
//     that the response is indistinguishable from a missing user or wrong password.
//     This preserves recovery for legitimate passwordless users without leaking
//     account state to unauthenticated callers.
//  3. Compare the provided password against the stored hash.
//  4. Return the user if the password matches; return an authentication error otherwise.
func (s *passwordMedImpl) Validate(ctx context.Context, identifier, password string) (*types.User, *apierror.APIError) {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.validate")
	defer span.End()

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, identifier)

	if err != nil {
		if apierror.IsNotFound(err) {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
		}
		return nil, tracing.Trace(span, err)
	}

	if user.HashedPassword == nil {
		if user.Email != nil {
			if err := s.sendPasswordResetEmail(ctx, user, nil); err != nil {
				return nil, tracing.Trace(span, err)
			}
		}
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
	}

	match, err := pwdutil.CompareHashAndPassword(ctx, *user.HashedPassword, password)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "password verification failed"))
	}

	if !match {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
	}

	return user, nil
}

// Update updates a user's password.
//
// 1. Hash the new password.
// 2. Persist the updated password hash in the repository.
// 3. Revoke all existing refresh tokens for the user.
// 4. Send a password updated notification email.
//
// Side effects:
//   - Updates the stored password hash.
//   - Revokes all refresh tokens for the user.
//   - Sends a password updated email.
func (s *passwordMedImpl) Update(ctx context.Context, user *types.User, newPassword string) *apierror.APIError {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.update")
	defer span.End()

	userRepo := s.repos.NewUserRepo()

	hashedPassword, err := pwdutil.HashPassword(ctx, newPassword)
	if err != nil {
		return err
	}

	err = userRepo.UpdatePassword(ctx, user.ID, hashedPassword)
	if err != nil {
		return err
	}

	err = s.refreshTokenMed.RevokeAll(ctx, user.ID)
	if err != nil {
		return err
	}

	// Notify the user that their password was changed
	if user.Email != nil {
		// Pass repos via context for the outbox publisher
		publishCtx := event.WithRepos(ctx, s.repos)
		s.notificationPublisher.PublishSendEmail(
			publishCtx,
			messaging.EmailSendData{
				To:         []string{*user.Email},
				Subject:    "Password Updated",
				TemplateID: constants.EmailTemplatePasswordUpdated,
				SentByID:   &user.ID,
			},
		)
	}

	return nil
}

// ValidatePasswordResetToken validates a password reset token and returns the associated user.
//
// 1. Decode and verify the JWT token as a password-reset type.
// 2. Look up the user by the token's subject (user ID).
// 3. Return an authentication error if the user is not found.
func (s *passwordMedImpl) ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *apierror.APIError) {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.validate_password_reset_token")
	defer span.End()

	claims, err := tokenutil.DecodeJWT(ctx, s.jwtSecret, token, tokenutil.JWTTypePasswordReset)
	if err != nil {
		return nil, err
	}

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, claims.Subject)

	if err != nil {
		if apierror.IsNotFound(err) {
			return nil, tracing.Trace(span, apierror.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
		}
		return nil, err
	}

	return user, nil
}

// RequestReset initiates a password reset flow for the given identifier.
//
//  1. Look up the user by identifier; silently succeed if not found to avoid leaking
//     information about registered identifiers.
//  2. Generate a short-lived password reset JWT (15 minutes).
//  3. Build the reset link, optionally scoped to an account slug.
//  4. Send a password reset email with the link.
//
// Behavior:
//   - Only returns an error for internal service failures; unknown identifiers
//     succeed silently to prevent enumeration.
//
// Side effects:
//   - Sends a password reset email.
func (s *passwordMedImpl) RequestReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.request_reset")
	defer span.End()

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, identifier)

	if err != nil {
		if apierror.IsNotFound(err) {
			return nil
		}
		if apierror.Is5XXErrorCode(err.Code) {
			return err
		}
		return nil
	}

	if user.Email == nil {
		return nil
	}

	return s.sendPasswordResetEmail(ctx, user, accountSlug)
}

// sendPasswordResetEmail mints a short-lived password reset JWT (15 minutes),
// builds a reset link optionally scoped to an account slug, and publishes the
// password reset email via the outbox. Caller must ensure user.Email is non-nil.
func (s *passwordMedImpl) sendPasswordResetEmail(ctx context.Context, user *types.User, accountSlug *string) *apierror.APIError {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.send_password_reset_email")
	defer span.End()

	resetToken, err := tokenutil.EncodeJWT(ctx, s.jwtSecret, user.ID, 15*time.Minute, tokenutil.JWTTypePasswordReset)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate password reset token."))
	}

	var resetLink string
	if accountSlug != nil && *accountSlug != "" {
		resetLink = fmt.Sprintf("%s/%s%s?t=%s", s.frontendURL, *accountSlug, constants.DashboardPathResetPassword, resetToken)
	} else {
		resetLink = fmt.Sprintf("%s%s?t=%s", s.frontendURL, constants.DashboardPathResetPassword, resetToken)
	}

	publishCtx := event.WithRepos(ctx, s.repos)
	s.notificationPublisher.PublishSendEmail(
		publishCtx,
		messaging.EmailSendData{
			To:         []string{*user.Email},
			Subject:    "Password Reset Request",
			TemplateID: constants.EmailTemplatePasswordReset,
			Params: map[string]any{
				"ResetLink":         resetLink,
				"ExpirationMinutes": 15,
			},
			SentByID: &user.ID,
		},
	)

	return nil
}
