package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var priorityRepoTracer = tracing.GetTracer("core-service.priority_repository")

type priorityRepoImpl struct {
	queries *sqlc.Queries
}

func NewPriorityRepo(queries *sqlc.Queries) domain.PriorityRepo {
	return &priorityRepoImpl{queries: queries}
}

func priorityCreatedAt(p *domain.Priority) time.Time { return p.CreatedAt }
func priorityID(p *domain.Priority) string           { return p.ID }

func buildPrioritySearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func mapPriorityRow(row sqlc.Priority) *domain.Priority {
	return &domain.Priority{
		ID:        row.ID,
		Name:      row.Name,
		Code:      constants.PriorityCode(row.Code),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *priorityRepoImpl) List(ctx context.Context, params domain.ListPrioritiesParams) (*domain.ListPrioritiesResult, *apierror.APIError) {
	ctx, span := priorityRepoTracer.Start(ctx, "repository.priority.list")
	defer span.End()

	searchQuery := buildPrioritySearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPrioritiesBackward(ctx, sqlc.ListPrioritiesBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			priorities := make([]*domain.Priority, len(rows))
			for i, row := range rows {
				priorities[i] = mapPriorityRow(row)
			}
			result, pageInfo := pagination.BuildPageString(priorities, params.Limit, cursorDir, priorityCreatedAt, priorityID)
			return &domain.ListPrioritiesResult{Priorities: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListPrioritiesForward(ctx, sqlc.ListPrioritiesForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		priorities := make([]*domain.Priority, len(rows))
		for i, row := range rows {
			priorities[i] = mapPriorityRow(row)
		}
		result, pageInfo := pagination.BuildPageString(priorities, params.Limit, cursorDir, priorityCreatedAt, priorityID)
		return &domain.ListPrioritiesResult{Priorities: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPrioritiesForward(ctx, sqlc.ListPrioritiesForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	priorities := make([]*domain.Priority, len(rows))
	for i, row := range rows {
		priorities[i] = mapPriorityRow(row)
	}
	result, pageInfo := pagination.BuildPageString(priorities, params.Limit, cursorDir, priorityCreatedAt, priorityID)
	return &domain.ListPrioritiesResult{Priorities: result, PageInfo: pageInfo}, nil
}

func (r *priorityRepoImpl) Get(ctx context.Context, identifier string) (*domain.Priority, *apierror.APIError) {
	ctx, span := priorityRepoTracer.Start(ctx, "repository.priority.get")
	defer span.End()

	row, err := r.queries.GetPriority(ctx, sqlc.GetPriorityParams{
		ID:   identifier,
		Code: identifier,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapPriorityRow(row), nil
}
