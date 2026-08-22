package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/trace"
)

var portalRegSessionRepoTracer = tracing.GetTracer("core-service.portal_registration_session_repository")

type portalRegistrationSessionRepoImpl struct {
	queries *sqlc.Queries
}

func NewPortalRegistrationSessionRepo(queries *sqlc.Queries) domain.PortalRegistrationSessionRepo {
	return &portalRegistrationSessionRepoImpl{queries: queries}
}

func (r *portalRegistrationSessionRepoImpl) Create(ctx context.Context, typeID string, params domain.CreatePortalRegistrationSessionParams) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.create")
	defer span.End()

	data, err := json.Marshal(params.SessionData)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal session data."))
	}

	if err := r.queries.CreatePortalRegistrationSession(ctx, sqlc.CreatePortalRegistrationSessionParams{
		TypeID:             typeID,
		UserID:             params.UserID,
		SellerAccountID:    params.SellerAccountID,
		SellerSlug:         params.SellerSlug,
		IsExistingCustomer: nullBool(params.IsExistingCustomer),
		Step:               string(params.Step),
		SessionData:        data,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.GetByTypeID(ctx, typeID)
}

func (r *portalRegistrationSessionRepoImpl) GetByTypeID(ctx context.Context, typeID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.get_by_type_id")
	defer span.End()

	row, err := r.queries.GetPortalRegistrationSessionByTypeID(ctx, typeID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return portalRegistrationSessionFromRow(row)
}

func (r *portalRegistrationSessionRepoImpl) GetIncomplete(ctx context.Context, userID, sellerAccountID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.get_incomplete")
	defer span.End()

	row, err := r.queries.GetIncompletePortalRegistrationSession(ctx, sqlc.GetIncompletePortalRegistrationSessionParams{
		UserID:          userID,
		SellerAccountID: sellerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return portalRegistrationSessionFromRow(row)
}

func (r *portalRegistrationSessionRepoImpl) Update(ctx context.Context, params domain.UpdatePortalRegistrationSessionParams) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.update")
	defer span.End()

	data, err := json.Marshal(params.SessionData)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal session data."))
	}

	if err := r.queries.UpdatePortalRegistrationSession(ctx, sqlc.UpdatePortalRegistrationSessionParams{
		Step:               string(params.Step),
		SessionData:        data,
		IsExistingCustomer: nullBool(params.IsExistingCustomer),
		TypeID:             params.TypeID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.GetByTypeID(ctx, params.TypeID)
}

func (r *portalRegistrationSessionRepoImpl) Complete(ctx context.Context, typeID, customerID string) (*domain.PortalRegistrationSession, *apierror.APIError) {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.complete")
	defer span.End()

	if err := r.queries.CompletePortalRegistrationSession(ctx, sqlc.CompletePortalRegistrationSessionParams{
		CustomerID: sql.NullString{String: customerID, Valid: customerID != ""},
		Step:       string(constants.PortalRegistrationStepCompleted),
		TypeID:     typeID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.GetByTypeID(ctx, typeID)
}

func (r *portalRegistrationSessionRepoImpl) Abandon(ctx context.Context, typeID string) *apierror.APIError {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.abandon")
	defer span.End()

	if err := r.queries.AbandonPortalRegistrationSession(ctx, typeID); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *portalRegistrationSessionRepoImpl) ListSessions(ctx context.Context, params domain.ListPortalRegistrationSessionsParams) (*domain.ListPortalRegistrationSessionsResult, *apierror.APIError) {
	ctx, span := portalRegSessionRepoTracer.Start(ctx, "repository.portal_registration_session.list")
	defer span.End()

	statusArg := portalRegistrationStatusArg(params.StatusFilter)
	searchArg := portalRegistrationSearchArg(params.SearchTerm)

	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor"))
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPortalRegistrationSessionsBackward(ctx, sqlc.ListPortalRegistrationSessionsBackwardParams{
				SellerAccountID: params.SellerAccountID,
				Status:          statusArg,
				Search:          searchArg,
				ExpiryThreshold: params.ExpiryThreshold,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return r.buildSessionPage(span, rows, params.Limit, cursorDir)
		}

		rows, err := r.queries.ListPortalRegistrationSessionsForward(ctx, sqlc.ListPortalRegistrationSessionsForwardParams{
			SellerAccountID: params.SellerAccountID,
			Status:          statusArg,
			Search:          searchArg,
			ExpiryThreshold: params.ExpiryThreshold,
			CursorCreatedAt: sql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        sql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return r.buildSessionPage(span, rows, params.Limit, cursorDir)
	}

	rows, err := r.queries.ListPortalRegistrationSessionsForward(ctx, sqlc.ListPortalRegistrationSessionsForwardParams{
		SellerAccountID: params.SellerAccountID,
		Status:          statusArg,
		Search:          searchArg,
		ExpiryThreshold: params.ExpiryThreshold,
		Limit:           params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return r.buildSessionPage(span, rows, params.Limit, cursorDir)
}

func (r *portalRegistrationSessionRepoImpl) buildSessionPage(span trace.Span, rows []sqlc.PortalRegistrationSession, limit int32, cursorDir *pagination.Direction) (*domain.ListPortalRegistrationSessionsResult, *apierror.APIError) {
	sessions := make([]*domain.PortalRegistrationSession, len(rows))
	for i, row := range rows {
		session, apiErr := portalRegistrationSessionFromRow(row)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		sessions[i] = session
	}
	result, pageInfo := pagination.BuildPageString(sessions, limit, cursorDir, portalRegistrationSessionCreatedAt, portalRegistrationSessionID)
	return &domain.ListPortalRegistrationSessionsResult{Sessions: result, PageInfo: pageInfo}, nil
}

func portalRegistrationSessionCreatedAt(s *domain.PortalRegistrationSession) time.Time {
	return s.CreatedAt
}
func portalRegistrationSessionID(s *domain.PortalRegistrationSession) string { return s.ID }

// portalRegistrationStatusArg maps the optional status filter to the sqlc interface{} narg: nil (no filter) becomes SQL NULL.
func portalRegistrationStatusArg(f *string) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

// portalRegistrationSearchArg maps the optional free-text search term to the sqlc interface{} narg: nil or empty (no search) becomes SQL NULL.
func portalRegistrationSearchArg(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func nullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

func portalRegistrationSessionFromRow(row sqlc.PortalRegistrationSession) (*domain.PortalRegistrationSession, *apierror.APIError) {
	var data domain.PortalRegistrationSessionData
	if len(row.SessionData) > 0 {
		if err := json.Unmarshal(row.SessionData, &data); err != nil {
			return nil, apierror.NewInternalError(err, "Failed to unmarshal session data.")
		}
	}

	s := &domain.PortalRegistrationSession{
		ID:              row.TypeID,
		UserID:          row.UserID,
		SellerAccountID: row.SellerAccountID,
		SellerSlug:      row.SellerSlug,
		Step:            constants.PortalRegistrationStep(row.Step),
		SessionData:     data,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.IsExistingCustomer.Valid {
		b := row.IsExistingCustomer.Bool
		s.IsExistingCustomer = &b
	}
	if row.CustomerID.Valid {
		c := row.CustomerID.String
		s.CustomerID = &c
	}
	if row.CompletedAt.Valid {
		t := row.CompletedAt.Time
		s.CompletedAt = &t
	}
	if row.AbandonedAt.Valid {
		t := row.AbandonedAt.Time
		s.AbandonedAt = &t
	}
	return s, nil
}
