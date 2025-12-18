package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	emailpkg "github.com/augno/api/services/auth-service/internal/email"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	pwdutil "github.com/augno/api/services/auth-service/internal/password"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
	tracing "github.com/augno/api/shared/tracing"
)

var passwordMedTracer = tracing.GetTracer("auth-service.password_mediator")

type passwordMedImpl struct {
	repos                 domain.RepoFactory
	refreshTokenMed       domain.RefreshTokenMed
	jwtUtils              domain.JWTUtils
	opaqueTokenUtils      domain.OpaqueTokenUtils
	notificationPublisher domain.NotificationPublisher
	templateRenderer      emailpkg.TemplateRenderer
	frontendURL           string
}

type PasswordMedConfig struct {
	Repos                 domain.RepoFactory
	RefreshTokenMed       domain.RefreshTokenMed
	JWTUtils              domain.JWTUtils
	OpaqueTokenUtils      domain.OpaqueTokenUtils
	NotificationPublisher domain.NotificationPublisher
	TemplateRenderer      emailpkg.TemplateRenderer
	FrontendURL           string
}

func NewPasswordMed(config PasswordMedConfig) domain.PasswordMed {
	if config.TemplateRenderer == nil {
		panic("TemplateRenderer is not set in the config.")
	}

	if config.NotificationPublisher == nil {
		panic("NotificationPublisher is not set in the config.")
	}

	if config.FrontendURL == "" {
		panic("FrontendURL is not set in the config.")
	}

	return &passwordMedImpl{
		repos:                 config.Repos,
		refreshTokenMed:       config.RefreshTokenMed,
		jwtUtils:              config.JWTUtils,
		opaqueTokenUtils:      config.OpaqueTokenUtils,
		notificationPublisher: config.NotificationPublisher,
		templateRenderer:      config.TemplateRenderer,
		frontendURL:           config.FrontendURL,
	}
}

func DefaultPasswordMedConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, notificationPublisher domain.NotificationPublisher, templateRenderer emailpkg.TemplateRenderer) PasswordMedConfig {
	factory := repository.NewRepoFactory(queries)
	return PasswordMedConfig{
		Repos:                 factory,
		RefreshTokenMed:       NewRefreshTokenMed(DefaultRefreshTokenMedConfig(queries, jwtSecret)),
		JWTUtils:              token.NewJWTUtils(token.DefaultJWTConfig(jwtSecret)),
		OpaqueTokenUtils:      token.NewOpaqueTokenUtils(token.DefaultOpaqueTokenConfig()),
		NotificationPublisher: notificationPublisher,
		TemplateRenderer:      templateRenderer,
		FrontendURL:           frontendURL,
	}
}

func NewDefaultPasswordMed(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, notificationPublisher domain.NotificationPublisher, templateRenderer emailpkg.TemplateRenderer) domain.PasswordMed {
	return NewPasswordMed(DefaultPasswordMedConfig(queries, jwtSecret, pepper, frontendURL, notificationPublisher, templateRenderer))
}

// Validate is a method that validates a user's password and returns the user if it is valid.
func (s *passwordMedImpl) Validate(ctx context.Context, identifier, password string) (*types.User, *contracts.APIError) {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.validate")
	defer span.End()

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, identifier)

	if err != nil || user == nil {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
	}

	match, err := pwdutil.CompareHashAndPassword(ctx, password, *user.HashedPassword)
	if err != nil {
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "password verification failed"))
	}

	if !match {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
	}

	return user, nil
}

// Update updates a user's password.
func (s *passwordMedImpl) Update(ctx context.Context, user *types.User, newPassword string) *contracts.APIError {
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
	if user.Email != nil && s.notificationPublisher != nil {
		var userName string
		if user.Name != nil {
			userName = *user.Name
		}
		body, err := s.templateRenderer.RenderPasswordUpdatedEmail(ctx, emailpkg.PasswordUpdatedEmailData{
			UserName: userName,
		})
		if err != nil {
			return tracing.Trace(span, err)
		}

		s.notificationPublisher.PublishSendEmail(
			ctx,
			[]string{*user.Email},
			"Password Updated",
			body,
			true,
			nil,
			user.ID,
			nil,
		)
	}

	return nil
}

// ValidatePasswordResetToken validates a password reset token and returns the user if it is valid.
func (s *passwordMedImpl) ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *contracts.APIError) {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.validate_password_reset_token")
	defer span.End()

	claims, err := s.jwtUtils.Decode(ctx, token, domain.JWTTypePasswordReset)
	if err != nil {
		return nil, err
	}

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, claims.Subject)

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, tracing.Trace(span, contracts.NewAuthenticationError(pwdutil.ErrInvalidCredentials))
	}

	return user, nil
}

// RequestReset handles the business logic for a request to reset a password.
// We only want to return an explicit error for internal service errors so we do not leak information about which identifiers
// are registered.
func (s *passwordMedImpl) RequestReset(ctx context.Context, identifier string, accountSlug *string) *contracts.APIError {
	ctx, span := passwordMedTracer.Start(ctx, "mediator.password.request_reset")
	defer span.End()

	userRepo := s.repos.NewUserRepo()
	user, err := userRepo.Find(ctx, identifier)

	if err != nil {
		if contracts.Is5XXErrorCode(err.Code) {
			return err
		}
		return nil
	}

	if user == nil || user.Email == nil {
		return nil
	}

	resetToken, err := s.jwtUtils.Encode(ctx, user.ID, 15*time.Minute, domain.JWTTypePasswordReset)
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to generate password reset token."))
	}

	var resetLink string
	if accountSlug != nil && *accountSlug != "" {
		resetLink = fmt.Sprintf("%s/%s/auth/password-reset?t=%s", s.frontendURL, *accountSlug, resetToken)
	} else {
		resetLink = fmt.Sprintf("%s/auth/password-reset?t=%s", s.frontendURL, resetToken)
	}

	body, err := s.templateRenderer.RenderPasswordResetEmail(ctx, emailpkg.PasswordResetEmailData{
		ResetLink:         resetLink,
		ExpirationMinutes: 15,
	})
	if err != nil {
		return tracing.Trace(span, err)
	}

	if s.notificationPublisher != nil {
		s.notificationPublisher.PublishSendEmail(
			ctx,
			[]string{*user.Email},
			"Password Reset Request",
			body,
			true,
			nil,
			user.ID,
			nil,
		)
	}

	return nil
}
