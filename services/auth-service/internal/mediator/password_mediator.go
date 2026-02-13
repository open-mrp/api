package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	pwdutil "github.com/augno/api/services/auth-service/internal/password"
	"github.com/augno/api/services/auth-service/internal/token"
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
	jwtUtils              domain.JWTUtils
	notificationPublisher domain.NotificationPublisher
	frontendURL           string
}

type PasswordMedConfig struct {
	Repos                 domain.RepoFactory
	RefreshTokenMed       domain.RefreshTokenMed
	JWTUtils              domain.JWTUtils
	NotificationPublisher domain.NotificationPublisher
	FrontendURL           string
}

// WithDefaults returns a new PasswordMedConfig with zero-value fields replaced by defaults.
func (c *PasswordMedConfig) WithDefaults() *PasswordMedConfig {
	if c == nil {
		c = &PasswordMedConfig{}
	}
	return &PasswordMedConfig{
		Repos:                 c.Repos,
		RefreshTokenMed:       c.RefreshTokenMed,
		JWTUtils:              c.JWTUtils,
		NotificationPublisher: c.NotificationPublisher,
		FrontendURL:           c.FrontendURL,
	}
}

func (c *PasswordMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("password mediator: repos is required")
	}
	if c.RefreshTokenMed == nil {
		return fmt.Errorf("password mediator: refresh token mediator is required")
	}
	if c.JWTUtils == nil {
		return fmt.Errorf("password mediator: jwt utils is required")
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
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &passwordMedImpl{
		repos:                 config.Repos,
		refreshTokenMed:       config.RefreshTokenMed,
		jwtUtils:              config.JWTUtils,
		notificationPublisher: config.NotificationPublisher,
		frontendURL:           config.FrontendURL,
	}
}

func DefaultPasswordMedConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, notificationPublisher domain.NotificationPublisher) *PasswordMedConfig {
	factory := repository.NewRepoFactory(queries)
	return &PasswordMedConfig{
		Repos:                 factory,
		RefreshTokenMed:       NewRefreshTokenMed(DefaultRefreshTokenMedConfig(queries, jwtSecret)),
		JWTUtils:              token.NewJWTUtils(&token.JWTConfig{Secret: jwtSecret}),
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
	}
}

func NewDefaultPasswordMed(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, notificationPublisher domain.NotificationPublisher) domain.PasswordMed {
	return NewPasswordMed(DefaultPasswordMedConfig(queries, jwtSecret, pepper, frontendURL, notificationPublisher))
}

// Validate is a method that validates a user's password and returns the user if it is valid.
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
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("user %s has no hashed password", user.ID)))
	}

	match, err := pwdutil.CompareHashAndPassword(ctx, password, *user.HashedPassword)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "password verification failed"))
	}

	if !match {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
	}

	return user, nil
}

// Update updates a user's password.
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

// ValidatePasswordResetToken validates a password reset token and returns the user if it is valid.
func (s *passwordMedImpl) ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *apierror.APIError) {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.validate_password_reset_token")
	defer span.End()

	claims, err := s.jwtUtils.Decode(ctx, token, domain.JWTTypePasswordReset)
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

// RequestReset handles the business logic for a request to reset a password.
// We only want to return an explicit error for internal service errors so we do not leak information about which identifiers
// are registered.
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

	resetToken, err := s.jwtUtils.Encode(ctx, user.ID, 15*time.Minute, domain.JWTTypePasswordReset)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate password reset token."))
	}

	var resetLink string
	if accountSlug != nil && *accountSlug != "" {
		resetLink = fmt.Sprintf("%s/%s/auth/password-reset?t=%s", s.frontendURL, *accountSlug, resetToken)
	} else {
		resetLink = fmt.Sprintf("%s/auth/password-reset?t=%s", s.frontendURL, resetToken)
	}

	// Pass repos via context for the outbox publisher
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
