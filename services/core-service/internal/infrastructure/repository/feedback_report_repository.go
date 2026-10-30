package repository

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var feedbackReportRepoTracer = tracing.GetTracer("core-service.feedback_report_repository")

type feedbackReportRepoImpl struct {
	queries *sqlc.Queries
}

func NewFeedbackReportRepo(queries *sqlc.Queries) domain.FeedbackReportRepo {
	return &feedbackReportRepoImpl{queries: queries}
}

func feedbackReportCreatedAt(r *domain.FeedbackReport) time.Time { return r.CreatedAt }
func feedbackReportID(r *domain.FeedbackReport) string           { return r.ID }

func mapFeedbackReportRow(row sqlc.FeedbackReport) *domain.FeedbackReport {
	return &domain.FeedbackReport{
		ID:        row.ID,
		AccountID: row.AccountID,
		UserID:    row.UserID,
		Message:   row.Message,
		Page:      db.StringFromNullString(row.Page),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func (r *feedbackReportRepoImpl) Create(ctx context.Context, feedbackReportID string, params domain.CreateFeedbackReportParams) (*domain.FeedbackReport, *apierror.APIError) {
	ctx, span := feedbackReportRepoTracer.Start(ctx, "repository.feedback_report.create")
	defer span.End()

	if err := r.queries.InsertFeedbackReport(ctx, sqlc.InsertFeedbackReportParams{
		ID:        feedbackReportID,
		AccountID: params.AccountID,
		UserID:    params.UserID,
		Message:   params.Message,
		Page:      toNullString(params.Page),
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, params.AccountID, feedbackReportID)
}

func (r *feedbackReportRepoImpl) Get(ctx context.Context, accountID, feedbackReportID string) (*domain.FeedbackReport, *apierror.APIError) {
	ctx, span := feedbackReportRepoTracer.Start(ctx, "repository.feedback_report.get")
	defer span.End()

	row, err := r.queries.GetFeedbackReport(ctx, sqlc.GetFeedbackReportParams{
		ID:        feedbackReportID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapFeedbackReportRow(row), nil
}

func (r *feedbackReportRepoImpl) List(ctx context.Context, params domain.ListFeedbackReportsParams) (*domain.ListFeedbackReportsResult, *apierror.APIError) {
	ctx, span := feedbackReportRepoTracer.Start(ctx, "repository.feedback_report.list")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	query, args := feedbackReportListPageQuery(params.AccountID, cursor, params.Limit+1)
	ids, err := selectStrings(ctx, r.queries.DB(), query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	byID := make(map[string]*domain.FeedbackReport, len(ids))
	if len(ids) > 0 {
		rows, err := r.queries.GetFeedbackReportsByIDs(ctx, sqlc.GetFeedbackReportsByIDsParams{
			Ids:       ids,
			AccountID: params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range rows {
			byID[row.ID] = mapFeedbackReportRow(row)
		}
	}

	result, pageInfo := pagination.BuildPageString(inPageOrder(ids, byID), params.Limit, cursorDirection(cursor), feedbackReportCreatedAt, feedbackReportID)
	return &domain.ListFeedbackReportsResult{FeedbackReports: result, PageInfo: pageInfo}, nil
}

func (r *feedbackReportRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.FeedbackReport, *apierror.APIError) {
	ctx, span := feedbackReportRepoTracer.Start(ctx, "repository.feedback_report.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetFeedbackReportsByIDs(ctx, sqlc.GetFeedbackReportsByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	reports := make([]*domain.FeedbackReport, len(rows))
	for i, row := range rows {
		reports[i] = mapFeedbackReportRow(row)
	}
	return reports, nil
}

func feedbackReportListPageQuery(accountID string, cursor *pagination.StringCursor, limit int32) (string, []any) {
	var f listFilter
	f.add("fr.account_id = ?", accountID)
	order := f.keyset(cursor, "fr.created_at", "fr.id")
	return "SELECT fr.id FROM feedback_report fr" + f.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", append(f.args, limit)
}
