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

var salesOrderStatusRepoTracer = tracing.GetTracer("core-service.sales_order_status_repository")

type salesOrderStatusRepoImpl struct {
	queries *sqlc.Queries
}

func NewSalesOrderStatusRepo(queries *sqlc.Queries) domain.SalesOrderStatusRepo {
	return &salesOrderStatusRepoImpl{queries: queries}
}

func salesOrderStatusCreatedAt(s *domain.SalesOrderStatus) time.Time { return s.CreatedAt }
func salesOrderStatusID(s *domain.SalesOrderStatus) string           { return s.ID }

func buildSalesOrderStatusSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapSalesOrderStatusForwardRow(row sqlc.ListSalesOrderStatusesForwardRow) *domain.SalesOrderStatus {
	return &domain.SalesOrderStatus{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapSalesOrderStatusBackwardRow(row sqlc.ListSalesOrderStatusesBackwardRow) *domain.SalesOrderStatus {
	return &domain.SalesOrderStatus{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *salesOrderStatusRepoImpl) List(ctx context.Context, params domain.ListSalesOrderStatusesParams) (*domain.ListSalesOrderStatusesResult, *apierror.APIError) {
	ctx, span := salesOrderStatusRepoTracer.Start(ctx, "repository.sales_order_status.list")
	defer span.End()

	searchQuery := buildSalesOrderStatusSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListSalesOrderStatusesBackward(ctx, sqlc.ListSalesOrderStatusesBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			statuses := make([]*domain.SalesOrderStatus, len(rows))
			for i, row := range rows {
				statuses[i] = mapSalesOrderStatusBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(statuses, params.Limit, cursorDir, salesOrderStatusCreatedAt, salesOrderStatusID)
			return &domain.ListSalesOrderStatusesResult{SalesOrderStatuses: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListSalesOrderStatusesForward(ctx, sqlc.ListSalesOrderStatusesForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		statuses := make([]*domain.SalesOrderStatus, len(rows))
		for i, row := range rows {
			statuses[i] = mapSalesOrderStatusForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(statuses, params.Limit, cursorDir, salesOrderStatusCreatedAt, salesOrderStatusID)
		return &domain.ListSalesOrderStatusesResult{SalesOrderStatuses: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListSalesOrderStatusesForward(ctx, sqlc.ListSalesOrderStatusesForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	statuses := make([]*domain.SalesOrderStatus, len(rows))
	for i, row := range rows {
		statuses[i] = mapSalesOrderStatusForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(statuses, params.Limit, cursorDir, salesOrderStatusCreatedAt, salesOrderStatusID)
	return &domain.ListSalesOrderStatusesResult{SalesOrderStatuses: result, PageInfo: pageInfo}, nil
}

func (r *salesOrderStatusRepoImpl) GetByIDs(ctx context.Context, ids []string) ([]*domain.SalesOrderStatus, *apierror.APIError) {
	ctx, span := salesOrderStatusRepoTracer.Start(ctx, "repository.sales_order_status.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetSalesOrderStatusesByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.SalesOrderStatus, len(rows))
	for i, row := range rows {
		out[i] = &domain.SalesOrderStatus{
			ID:        row.ID,
			Code:      row.Code,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}
