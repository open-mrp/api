package repository

import (
	"context"
	gosql "database/sql"
	"encoding/json"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var registrationSessionRepoTracer = tracing.GetTracer("auth-service.registration_session_repository")

type registrationSessionRepoImpl struct {
	queries *sqlc.Queries
}

func NewRegistrationSessionRepo(queries *sqlc.Queries) domain.RegistrationSessionRepo {
	return &registrationSessionRepoImpl{queries: queries}
}

func (r *registrationSessionRepoImpl) GetByEmail(ctx context.Context, email string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.get_by_email")
	defer span.End()

	row, err := r.queries.GetRegistrationSessionByEmail(ctx, email)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return mapRegistrationSession(row), nil
}

func (r *registrationSessionRepoImpl) GetByTypeID(ctx context.Context, typeID string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.get_by_type_id")
	defer span.End()

	row, err := r.queries.GetRegistrationSessionByTypeID(ctx, typeID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return mapRegistrationSession(row), nil
}

func (r *registrationSessionRepoImpl) Create(ctx context.Context, session *domain.RegistrationSession) (int64, *apierror.APIError) {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.create")
	defer span.End()

	sessionData, err := json.Marshal(session.SessionData)
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal session data."))
	}

	result, sqlErr := r.queries.CreateRegistrationSession(ctx, sqlc.CreateRegistrationSessionParams{
		TypeID:            session.TypeID,
		Email:             session.Email,
		PlanCode:          session.PlanCode,
		Step:              string(session.Step),
		VerificationToken: session.VerificationToken,
		IsEmailVerified:   session.IsEmailVerified,
		IsExistingUser:    nullBoolFromBoolPtr(session.IsExistingUser),
		SessionData:       sessionData,
	})
	if apiErr := db.MapSQLError(sqlErr); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to get last insert ID."))
	}

	return id, nil
}

func (r *registrationSessionRepoImpl) UpdatePlanCode(ctx context.Context, id int64, planCode string) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_plan_code")
	defer span.End()

	err := r.queries.UpdateRegistrationSessionPlanCode(ctx, sqlc.UpdateRegistrationSessionPlanCodeParams{
		PlanCode: planCode,
		ID:       id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationSessionRepoImpl) UpdateToken(ctx context.Context, id int64, verificationToken string) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_token")
	defer span.End()

	err := r.queries.UpdateRegistrationSessionToken(ctx, sqlc.UpdateRegistrationSessionTokenParams{
		VerificationToken: verificationToken,
		ID:                id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func nullBoolFromBoolPtr(b *bool) gosql.NullBool {
	if b == nil {
		return gosql.NullBool{Valid: false}
	}
	return gosql.NullBool{Bool: *b, Valid: true}
}

func boolPtrFromNullBool(nb gosql.NullBool) *bool {
	if !nb.Valid {
		return nil
	}
	return &nb.Bool
}

func (r *registrationSessionRepoImpl) GetByToken(ctx context.Context, token string) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.get_by_token")
	defer span.End()

	row, err := r.queries.GetRegistrationSessionByToken(ctx, token)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return mapRegistrationSession(row), nil
}

func (r *registrationSessionRepoImpl) GetByID(ctx context.Context, id int64) (*domain.RegistrationSession, *apierror.APIError) {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.get_by_id")
	defer span.End()

	row, err := r.queries.GetRegistrationSessionByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return mapRegistrationSession(row), nil
}

func (r *registrationSessionRepoImpl) UpdateEmailVerified(ctx context.Context, id int64, isExistingUser *bool) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_email_verified")
	defer span.End()

	err := r.queries.UpdateRegistrationSessionEmailVerified(ctx, sqlc.UpdateRegistrationSessionEmailVerifiedParams{
		IsEmailVerified: true,
		IsExistingUser:  nullBoolFromBoolPtr(isExistingUser),
		ID:              id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationSessionRepoImpl) UpdateStep(ctx context.Context, id int64, step constants.RegistrationStep, sessionData domain.RegistrationSessionData) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_step")
	defer span.End()

	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal session data."))
	}

	sqlErr := r.queries.UpdateRegistrationSessionStep(ctx, sqlc.UpdateRegistrationSessionStepParams{
		Step:        string(step),
		SessionData: sessionDataJSON,
		ID:          id,
	})
	if apiErr := db.MapSQLError(sqlErr); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationSessionRepoImpl) UpdateUser(ctx context.Context, id int64, userID string, sessionData domain.RegistrationSessionData) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_user")
	defer span.End()

	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal session data."))
	}

	sqlErr := r.queries.UpdateRegistrationSessionUser(ctx, sqlc.UpdateRegistrationSessionUserParams{
		UserID:      gosql.NullString{String: userID, Valid: true},
		SessionData: sessionDataJSON,
		ID:          id,
	})
	if apiErr := db.MapSQLError(sqlErr); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationSessionRepoImpl) UpdateStripeCustomer(ctx context.Context, id int64, stripeCustomerID *string, stripeCheckoutSessionID *string) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_stripe_customer")
	defer span.End()

	err := r.queries.UpdateRegistrationSessionStripeCustomer(ctx, sqlc.UpdateRegistrationSessionStripeCustomerParams{
		StripeCustomerID:        db.NullStringPtr(stripeCustomerID),
		StripeCheckoutSessionID: db.NullStringPtr(stripeCheckoutSessionID),
		ID:                      id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationSessionRepoImpl) UpdatePaymentCompleted(ctx context.Context, id int64, paymentCompleted bool, stripeSubscriptionID *string) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.update_payment_completed")
	defer span.End()

	err := r.queries.UpdateRegistrationSessionPaymentCompleted(ctx, sqlc.UpdateRegistrationSessionPaymentCompletedParams{
		PaymentCompleted:     paymentCompleted,
		StripeSubscriptionID: db.NullStringPtr(stripeSubscriptionID),
		ID:                   id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *registrationSessionRepoImpl) Complete(ctx context.Context, id int64, accountID *string) *apierror.APIError {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.complete")
	defer span.End()

	err := r.queries.CompleteRegistrationSession(ctx, sqlc.CompleteRegistrationSessionParams{
		AccountID: db.NullStringPtr(accountID),
		ID:        id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func regSessionCreatedAt(s *domain.RegistrationSession) time.Time { return s.CreatedAt }
func regSessionID(s *domain.RegistrationSession) int64            { return s.ID }

func (r *registrationSessionRepoImpl) ListByUserID(ctx context.Context, userID string, cursor *string, limit int32) ([]*domain.RegistrationSession, pagination.PageInfo, *apierror.APIError) {
	ctx, span := registrationSessionRepoTracer.Start(ctx, "repository.registration_session.list_by_user_id")
	defer span.End()

	var cursorDir *pagination.Direction

	if cursor != nil {
		cur, err := pagination.DecodeCursor(*cursor)
		if err != nil {
			return nil, pagination.PageInfo{}, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListRegistrationSessionsByUserIDBackward(ctx, sqlc.ListRegistrationSessionsByUserIDBackwardParams{
				UserID:          gosql.NullString{String: userID, Valid: true},
				CursorCreatedAt: cur.CreatedAt,
				CursorID:        cur.ID,
				Limit:           limit + 1,
			})
			if err != nil {
				return nil, pagination.PageInfo{}, tracing.Trace(span, apierror.NewInternalError(err, "Failed to list registration sessions."))
			}
			sessions := make([]*domain.RegistrationSession, len(rows))
			for i, row := range rows {
				sessions[i] = mapRegistrationSession(row)
			}
			result, pageInfo := pagination.BuildPage(sessions, limit, cursorDir, regSessionCreatedAt, regSessionID)
			return result, pageInfo, nil
		}

		// Forward
		rows, err := r.queries.ListRegistrationSessionsByUserIDForward(ctx, sqlc.ListRegistrationSessionsByUserIDForwardParams{
			UserID:          gosql.NullString{String: userID, Valid: true},
			CursorCreatedAt: gosql.NullTime{Time: cur.CreatedAt, Valid: true},
			CursorID:        gosql.NullInt64{Int64: cur.ID, Valid: true},
			Limit:           limit + 1,
		})
		if err != nil {
			return nil, pagination.PageInfo{}, tracing.Trace(span, apierror.NewInternalError(err, "Failed to list registration sessions."))
		}
		sessions := make([]*domain.RegistrationSession, len(rows))
		for i, row := range rows {
			sessions[i] = mapRegistrationSession(row)
		}
		result, pageInfo := pagination.BuildPage(sessions, limit, cursorDir, regSessionCreatedAt, regSessionID)
		return result, pageInfo, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListRegistrationSessionsByUserIDForward(ctx, sqlc.ListRegistrationSessionsByUserIDForwardParams{
		UserID: gosql.NullString{String: userID, Valid: true},
		Limit:  limit + 1,
	})
	if err != nil {
		return nil, pagination.PageInfo{}, tracing.Trace(span, apierror.NewInternalError(err, "Failed to list registration sessions."))
	}

	sessions := make([]*domain.RegistrationSession, len(rows))
	for i, row := range rows {
		sessions[i] = mapRegistrationSession(row)
	}
	result, pageInfo := pagination.BuildPage(sessions, limit, cursorDir, regSessionCreatedAt, regSessionID)
	return result, pageInfo, nil
}

func mapRegistrationSession(row sqlc.RegistrationSession) *domain.RegistrationSession {
	var sessionData domain.RegistrationSessionData
	if len(row.SessionData) > 0 {
		_ = json.Unmarshal(row.SessionData, &sessionData)
	}

	return &domain.RegistrationSession{
		ID:                      row.ID,
		TypeID:                  row.TypeID,
		Email:                   row.Email,
		PlanCode:                row.PlanCode,
		Step:                    constants.RegistrationStep(row.Step),
		VerificationToken:       row.VerificationToken,
		IsEmailVerified:         row.IsEmailVerified,
		IsExistingUser:          boolPtrFromNullBool(row.IsExistingUser),
		UserID:                  db.StringFromNullString(row.UserID),
		AccountID:               db.StringFromNullString(row.AccountID),
		StripeCustomerID:        db.StringFromNullString(row.StripeCustomerID),
		StripeCheckoutSessionID: db.StringFromNullString(row.StripeCheckoutSessionID),
		StripeSubscriptionID:    db.StringFromNullString(row.StripeSubscriptionID),
		PaymentCompleted:        row.PaymentCompleted,
		SessionData:             sessionData,
		CompletedAt:             db.TimeFromNullTime(row.CompletedAt),
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}
