package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	pwdutil "github.com/augno/api/services/auth-service/internal/password"
	tokenutil "github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

const (
	// registrationSessionTTL is the maximum age of a registration session
	// before it is considered expired and ignored.
	registrationSessionTTL = 7 * 24 * time.Hour // 7 days

	// verificationTokenTTL is the maximum age of a verification token
	// before it is considered expired.
	verificationTokenTTL = 24 * time.Hour // 24 hours
)

var registrationMedTracer = tracing.GetTracer("auth-service.registration_mediator")

type registrationMedImpl struct {
	repos                 domain.RepoFactory
	notificationPublisher domain.NotificationPublisher
	frontendURL           string
	userMed               domain.UserMed
	refreshTokenMed       domain.RefreshTokenMed
}

type RegistrationMedConfig struct {
	Repos                 domain.RepoFactory
	NotificationPublisher domain.NotificationPublisher
	FrontendURL           string
	UserMed               domain.UserMed
	RefreshTokenMed       domain.RefreshTokenMed
}

func (c *RegistrationMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("registration mediator: repos is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("registration mediator: notification publisher is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("registration mediator: frontend url is required")
	}
	if c.UserMed == nil {
		return fmt.Errorf("registration mediator: user mediator is required")
	}
	if c.RefreshTokenMed == nil {
		return fmt.Errorf("registration mediator: refresh token mediator is required")
	}
	return nil
}

func NewRegistrationMed(config *RegistrationMedConfig) domain.RegistrationMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &registrationMedImpl{
		repos:                 config.Repos,
		notificationPublisher: config.NotificationPublisher,
		frontendURL:           config.FrontendURL,
		userMed:               config.UserMed,
		refreshTokenMed:       config.RefreshTokenMed,
	}
}

// CreateSession creates a new registration session or returns an existing active session
// for the given email (idempotent).
//
//  1. Check if the user already exists (noted but does not prevent session creation).
//  2. Look for an existing non-expired session for the email; if found, update the plan code
//     if different and resend the verification email.
//  3. Generate a unique type ID and verification token.
//  4. Create a new registration session record.
//  5. Send the verification email.
//
// Side effects:
//   - Sends a verification email to the user.
func (m *registrationMedImpl) CreateSession(ctx context.Context, input domain.CreateRegistrationSessionInput) (*domain.CreateRegistrationSessionResult, *apierror.APIError) {
	ctx, span := registrationMedTracer.Start(ctx, "mediator.registration.create_session")
	defer span.End()

	userRepo := m.repos.NewUserRepo()
	regSessionRepo := m.repos.NewRegistrationSessionRepo()

	// Check if user already exists (note but don't fail)
	existingUser, _ := userRepo.Find(ctx, input.Email)
	isExistingUser := existingUser != nil

	// Return existing active session if one exists
	existingSession, err := regSessionRepo.GetByEmail(ctx, input.Email)
	if err != nil && !apierror.IsNotFound(err) {
		return nil, tracing.Trace(span, err)
	}
	if existingSession != nil {
		// Ignore expired sessions — treat as if none exists
		if time.Since(existingSession.CreatedAt) <= registrationSessionTTL {
			// Update plan code if the client requested a different plan
			if existingSession.PlanCode != input.PlanCode {
				if updateErr := regSessionRepo.UpdatePlanCode(ctx, existingSession.ID, input.PlanCode); updateErr != nil {
					return nil, tracing.Trace(span, updateErr)
				}
			}
			m.sendVerificationEmail(ctx, input.Email, existingSession.VerificationToken, false)
			return &domain.CreateRegistrationSessionResult{
				SessionID: existingSession.TypeID,
			}, nil
		}
	}

	// Generate type ID and verification token
	typeID, genErr := id.GenID(id.RegistrationFlowIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	verificationToken, tokenErr := tokenutil.GenOpaqueToken(ctx)
	if tokenErr != nil {
		return nil, tracing.Trace(span, tokenErr)
	}

	// Create registration session
	session := &domain.RegistrationSession{
		TypeID:            typeID,
		Email:             input.Email,
		PlanCode:          input.PlanCode,
		Step:              constants.RegistrationStepVerification,
		VerificationToken: verificationToken,
		IsEmailVerified:   false,
		IsExistingUser:    &isExistingUser,
	}

	_, createErr := regSessionRepo.Create(ctx, session)
	if createErr != nil {
		return nil, tracing.Trace(span, createErr)
	}

	m.sendVerificationEmail(ctx, input.Email, verificationToken, isExistingUser)

	return &domain.CreateRegistrationSessionResult{
		SessionID: typeID,
	}, nil
}

// ResendVerificationEmail regenerates the verification token and resends the verification email.
//
// 1. Look up the session by type ID.
// 2. Validate the session is not completed and email is not already verified.
// 3. Generate a new verification token and update the session.
// 4. Send the verification email with the new token.
//
// Side effects:
//   - Rotates the verification token.
//   - Sends a new verification email.
func (m *registrationMedImpl) ResendVerificationEmail(ctx context.Context, sessionID string) *apierror.APIError {
	ctx, span := registrationMedTracer.Start(ctx, "mediator.registration.resend_verification_email")
	defer span.End()

	regSessionRepo := m.repos.NewRegistrationSessionRepo()

	session, err := regSessionRepo.GetByTypeID(ctx, sessionID)
	if err != nil {
		return tracing.Trace(span, err)
	}

	if session.CompletedAt != nil {
		return tracing.Trace(span, apierror.NewValidationError("Registration session is already completed."))
	}

	if session.IsEmailVerified {
		return tracing.Trace(span, apierror.NewValidationError("Email is already verified."))
	}

	// Rotate verification token
	newToken, tokenErr := tokenutil.GenOpaqueToken(ctx)
	if tokenErr != nil {
		return tracing.Trace(span, tokenErr)
	}

	if updateErr := regSessionRepo.UpdateToken(ctx, session.ID, newToken); updateErr != nil {
		return tracing.Trace(span, updateErr)
	}

	m.sendVerificationEmail(ctx, session.Email, newToken, false)

	return nil
}

// VerifyToken verifies the email verification token and marks the session as email-verified.
//
// 1. Look up the session by verification token.
// 2. Reject completed sessions.
// 3. Check token expiry (24-hour TTL from last update).
// 4. If already verified, return the current session without changes (idempotent).
// 5. Check if a user already exists for the session's email.
// 6. Mark the email as verified and advance the step to user_details.
// 7. Re-fetch and return the updated session.
func (m *registrationMedImpl) VerifyToken(ctx context.Context, token string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationMedTracer.Start(ctx, "mediator.registration.verify_token")
	defer span.End()

	regSessionRepo := m.repos.NewRegistrationSessionRepo()
	userRepo := m.repos.NewUserRepo()

	// Look up session by verification token
	session, err := regSessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, tracing.Trace(span, err)
	}

	// Already completed sessions cannot be verified
	if session.CompletedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Registration session is already completed."))
	}

	// Check token expiry
	if time.Since(session.UpdatedAt) > verificationTokenTTL {
		return nil, tracing.Trace(span, apierror.NewValidationError("Verification token has expired. Please request a new one."))
	}

	// If already verified, return current session
	if session.IsEmailVerified {
		refreshed, refreshErr := regSessionRepo.GetByID(ctx, session.ID)
		if refreshErr != nil {
			return nil, tracing.Trace(span, refreshErr)
		}
		return refreshed, nil
	}

	// Check if user already exists
	existingUser, _ := userRepo.Find(ctx, session.Email)
	isExistingUser := existingUser != nil

	// Mark email as verified
	if updateErr := regSessionRepo.UpdateEmailVerified(ctx, session.ID, &isExistingUser); updateErr != nil {
		return nil, tracing.Trace(span, updateErr)
	}

	// Advance step to user_details
	if stepErr := regSessionRepo.UpdateStep(ctx, session.ID, constants.RegistrationStepUserDetails, session.SessionData); stepErr != nil {
		return nil, tracing.Trace(span, stepErr)
	}

	// Re-fetch with fresh timestamps
	result, fetchErr := regSessionRepo.GetByID(ctx, session.ID)
	if fetchErr != nil {
		return nil, tracing.Trace(span, fetchErr)
	}

	return result, nil
}

// CreateUserForSession creates or resolves a user for the registration session and returns
// the user ID with auth tokens.
//
// 1. Look up the session by type ID and validate it is not completed and email is verified.
// 2. If the session already has a user, generate tokens for the existing user (idempotent).
// 3. Check if a user already exists with the session's email; reuse if so.
// 4. Otherwise, hash the password and create a new user record.
// 5. Associate the user with the session and update session data with the user name.
// 6. Advance the session step to account_details.
// 7. Generate and return an access token and refresh token.
//
// Side effects:
//   - May create a new user record.
//   - Associates the user with the session.
//   - Advances the session step to account_details.
func (m *registrationMedImpl) CreateUserForSession(ctx context.Context, input domain.CreateUserForRegistrationInput) (*domain.CreateUserForRegistrationOutput, *apierror.APIError) {
	ctx, span := registrationMedTracer.Start(ctx, "mediator.registration.create_user_for_session")
	defer span.End()

	regSessionRepo := m.repos.NewRegistrationSessionRepo()

	session, err := regSessionRepo.GetByTypeID(ctx, input.SessionID)
	if err != nil {
		return nil, tracing.Trace(span, err)
	}

	if session.CompletedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Registration session is already completed."))
	}

	if !session.IsEmailVerified {
		return nil, tracing.Trace(span, apierror.NewValidationError("Email must be verified before creating user."))
	}

	// If user already exists on session, generate tokens and return
	if session.UserID != nil {
		accessToken, tokenErr := m.userMed.GenAuthAccessToken(ctx, *session.UserID)
		if tokenErr != nil {
			return nil, tracing.Trace(span, tokenErr)
		}

		refreshTokenModel, rtErr := m.refreshTokenMed.Create(ctx, *session.UserID, nil)
		if rtErr != nil {
			return nil, tracing.Trace(span, rtErr)
		}

		return &domain.CreateUserForRegistrationOutput{
			UserID:       *session.UserID,
			AccessToken:  accessToken,
			RefreshToken: refreshTokenModel.Token,
		}, nil
	}

	// Check if a user already exists with this email
	userRepo := m.repos.NewUserRepo()
	existingUser, findErr := userRepo.Find(ctx, session.Email)
	if findErr != nil && !apierror.IsNotFound(findErr) {
		return nil, tracing.Trace(span, findErr)
	}

	var userID string
	if existingUser != nil {
		userID = existingUser.ID
	} else {
		// Hash password and create user
		hashedPassword, hashErr := pwdutil.HashPassword(ctx, input.Password)
		if hashErr != nil {
			return nil, tracing.Trace(span, hashErr)
		}

		newUserID, genErr := id.GenID(id.UserIDPrefix, nil)
		if genErr != nil {
			return nil, tracing.Trace(span, genErr)
		}
		userID = newUserID

		_, createErr := userRepo.Create(ctx, userID, session.Email, input.Name, hashedPassword)
		if createErr != nil {
			return nil, tracing.Trace(span, createErr)
		}
	}

	// Update session with user ID and name
	sessionData := session.SessionData
	sessionData.UserName = input.Name
	if updateErr := regSessionRepo.UpdateUser(ctx, session.ID, userID, sessionData); updateErr != nil {
		return nil, tracing.Trace(span, updateErr)
	}

	// Advance step to account_details
	if stepErr := regSessionRepo.UpdateStep(ctx, session.ID, constants.RegistrationStepAccountDetails, sessionData); stepErr != nil {
		return nil, tracing.Trace(span, stepErr)
	}

	// Generate auth tokens
	accessToken, tokenErr := m.userMed.GenAuthAccessToken(ctx, userID)
	if tokenErr != nil {
		return nil, tracing.Trace(span, tokenErr)
	}

	refreshTokenModel, rtErr := m.refreshTokenMed.Create(ctx, userID, nil)
	if rtErr != nil {
		return nil, tracing.Trace(span, rtErr)
	}

	return &domain.CreateUserForRegistrationOutput{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenModel.Token,
	}, nil
}

func (m *registrationMedImpl) getSessionByTypeID(ctx context.Context, sessionID string) (*domain.RegistrationSession, *apierror.APIError) {
	regSessionRepo := m.repos.NewRegistrationSessionRepo()
	return regSessionRepo.GetByTypeID(ctx, sessionID)
}

// UpdateSession updates an in-progress registration session's step and form data.
//
// 1. Look up the session by type ID and validate it is not completed.
// 2. Validate the step transition allows only forward progression.
// 3. Merge the provided session data into the existing data (non-nil fields only).
// 4. Persist the updated step and data.
// 5. Re-fetch and return the refreshed session.
func (m *registrationMedImpl) UpdateSession(ctx context.Context, sessionID string, step *constants.RegistrationStep, sessionData *domain.UpdateRegistrationSessionData) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationMedTracer.Start(ctx, "mediator.registration.update_session")
	defer span.End()

	regSessionRepo := m.repos.NewRegistrationSessionRepo()

	session, err := m.getSessionByTypeID(ctx, sessionID)
	if err != nil {
		return nil, tracing.Trace(span, err)
	}

	if session.CompletedAt != nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Registration session is already completed."))
	}

	// Validate step transition — only allow forward progression
	targetStep := session.Step
	if step != nil {
		if !step.IsValid() {
			return nil, tracing.Trace(span, apierror.NewValidationError("Invalid registration step."))
		}
		if !step.IsAfter(session.Step) {
			return nil, tracing.Trace(span, apierror.NewValidationError("Cannot move to a previous registration step."))
		}
		targetStep = *step
	}

	// Merge session data if provided — only overwrite fields that are non-nil
	mergedData := session.SessionData
	if sessionData != nil {
		sessionData.MergeInto(&mergedData)
	}
	if step != nil || sessionData != nil {
		if stepErr := regSessionRepo.UpdateStep(ctx, session.ID, targetStep, mergedData); stepErr != nil {
			return nil, tracing.Trace(span, stepErr)
		}
	}

	// Re-fetch with fresh timestamps
	result, fetchErr := regSessionRepo.GetByID(ctx, session.ID)
	if fetchErr != nil {
		return nil, tracing.Trace(span, fetchErr)
	}

	return result, nil
}

// GetSession returns the registration session for the given type ID.
//
// 1. Look up and return the session by its type ID.
func (m *registrationMedImpl) GetSession(ctx context.Context, sessionID string) (*domain.RegistrationSession, *apierror.APIError) {
	return m.getSessionByTypeID(ctx, sessionID)
}

// CompleteSession marks a registration session as completed and records the account ID.
//
// 1. Look up the session by type ID.
// 2. Mark the session as completed with the provided account ID in the repository.
func (m *registrationMedImpl) CompleteSession(ctx context.Context, sessionID, accountID string) *apierror.APIError {
	ctx, span := registrationMedTracer.Start(ctx, "mediator.registration.complete_session")
	defer span.End()

	session, apiErr := m.getSessionByTypeID(ctx, sessionID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	regSessionRepo := m.repos.NewRegistrationSessionRepo()
	return regSessionRepo.Complete(ctx, session.ID, &accountID)
}

func (m *registrationMedImpl) sendVerificationEmail(ctx context.Context, email, token string, isExistingUser bool) {
	templateID := constants.EmailTemplateRegistrationVerify
	verifyLink := fmt.Sprintf("%s%s?t=%s", m.frontendURL, constants.DashboardPathRegisterVerify, token)

	if isExistingUser {
		templateID = constants.EmailTemplateRegistrationVerifyExisting
		verifyLink = fmt.Sprintf("%s%s", m.frontendURL, constants.DashboardPathLogin)
	}

	publishCtx := event.WithRepos(ctx, m.repos)
	m.notificationPublisher.PublishSendEmail(
		publishCtx,
		messaging.EmailSendData{
			To:         []string{email},
			Subject:    "Verify your email",
			TemplateID: templateID,
			Params: map[string]any{
				"VerifyURL": verifyLink,
			},
		},
	)
}
