package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var accountStatusRepoTracer = tracing.GetTracer("core-service.account_status_repository")

type accountStatusRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountStatusRepo(queries *sqlc.Queries) domain.AccountStatusRepo {
	return &accountStatusRepoImpl{queries: queries}
}

func accountStatusCreatedAt(as *domain.AccountStatus) time.Time { return as.CreatedAt }
func accountStatusID(as *domain.AccountStatus) string           { return as.ID }

func buildAccountStatusSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapAccountStatusRow(row sqlc.AccountStatus) *domain.AccountStatus {
	return &domain.AccountStatus{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *accountStatusRepoImpl) List(ctx context.Context, params domain.ListAccountStatusesParams) (*domain.ListAccountStatusesResult, *apierror.APIError) {
	ctx, span := accountStatusRepoTracer.Start(ctx, "repository.account_status.list")
	defer span.End()

	searchQuery := buildAccountStatusSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAccountStatusesBackward(ctx, sqlc.ListAccountStatusesBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			statuses := make([]*domain.AccountStatus, len(rows))
			for i, row := range rows {
				statuses[i] = mapAccountStatusRow(row)
			}
			result, pageInfo := pagination.BuildPageString(statuses, params.Limit, cursorDir, accountStatusCreatedAt, accountStatusID)
			return &domain.ListAccountStatusesResult{AccountStatuses: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListAccountStatusesForward(ctx, sqlc.ListAccountStatusesForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		statuses := make([]*domain.AccountStatus, len(rows))
		for i, row := range rows {
			statuses[i] = mapAccountStatusRow(row)
		}
		result, pageInfo := pagination.BuildPageString(statuses, params.Limit, cursorDir, accountStatusCreatedAt, accountStatusID)
		return &domain.ListAccountStatusesResult{AccountStatuses: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListAccountStatusesForward(ctx, sqlc.ListAccountStatusesForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	statuses := make([]*domain.AccountStatus, len(rows))
	for i, row := range rows {
		statuses[i] = mapAccountStatusRow(row)
	}
	result, pageInfo := pagination.BuildPageString(statuses, params.Limit, cursorDir, accountStatusCreatedAt, accountStatusID)
	return &domain.ListAccountStatusesResult{AccountStatuses: result, PageInfo: pageInfo}, nil
}

func (r *accountStatusRepoImpl) Get(ctx context.Context, identifier string) (*domain.AccountStatus, *apierror.APIError) {
	ctx, span := accountStatusRepoTracer.Start(ctx, "repository.account_status.get")
	defer span.End()

	row, err := r.queries.GetAccountStatus(ctx, sqlc.GetAccountStatusParams{
		ID:   identifier,
		Code: identifier,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapAccountStatusRow(row), nil
}

func (r *accountStatusRepoImpl) GetByIDs(ctx context.Context, ids []string) ([]*domain.AccountStatus, *apierror.APIError) {
	ctx, span := accountStatusRepoTracer.Start(ctx, "repository.account_status.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetAccountStatusesByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.AccountStatus, len(rows))
	for i, row := range rows {
		out[i] = mapAccountStatusRow(row)
	}
	return out, nil
}
