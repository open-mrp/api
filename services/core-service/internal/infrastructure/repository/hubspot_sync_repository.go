package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var hubspotSyncRepoTracer = tracing.GetTracer("core-service.hubspot_sync_repository")

type hubspotSyncRepoImpl struct {
	queries *sqlc.Queries
}

func NewHubspotSyncRepo(queries *sqlc.Queries) domain.HubspotSyncRepo {
	return &hubspotSyncRepoImpl{queries: queries}
}

// --- Jobs ---

func (r *hubspotSyncRepoImpl) CreateJob(ctx context.Context, params domain.CreateHubspotSyncJobParams) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.create_job")
	defer span.End()

	jobID, apiErr := id.GenID(id.HubspotSyncJobIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertHubspotSyncJob(ctx, sqlc.InsertHubspotSyncJobParams{
		ID:             jobID,
		AccountID:      params.AccountID,
		Status:         params.Status,
		GoliveCutoffAt: db.NullTimePtr(params.GoLiveCutoffAt),
	}); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return r.GetJob(ctx, params.AccountID, jobID)
}

func (r *hubspotSyncRepoImpl) GetJob(ctx context.Context, accountID, jobID string) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.get_job")
	defer span.End()

	row, err := r.queries.GetHubspotSyncJob(ctx, sqlc.GetHubspotSyncJobParams{ID: jobID, AccountID: accountID})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return mapJob(row), nil
}

// GetLatestJobForAccount returns the account's most recent job, or (nil, nil) when none exist.
func (r *hubspotSyncRepoImpl) GetLatestJobForAccount(ctx context.Context, accountID string) (*domain.HubspotSyncJob, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.get_latest_job")
	defer span.End()

	row, err := r.queries.GetLatestHubspotSyncJobForAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return mapJob(row), nil
}

func (r *hubspotSyncRepoImpl) UpdateJob(ctx context.Context, params domain.UpdateHubspotSyncJobParams) *apierror.APIError {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.update_job")
	defer span.End()

	if _, err := r.queries.UpdateHubspotSyncJob(ctx, sqlc.UpdateHubspotSyncJobParams{
		ID:          params.ID,
		AccountID:   params.AccountID,
		Status:      db.NullStringPtr(params.Status),
		Cursors:     db.NullableRawMessage(params.Cursors),
		Counts:      db.NullableRawMessage(params.Counts),
		LastError:   db.NullStringPtr(params.LastError),
		StartedAt:   db.NullTimePtr(params.StartedAt),
		CompletedAt: db.NullTimePtr(params.CompletedAt),
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *hubspotSyncRepoImpl) ClaimJobForExecute(ctx context.Context, accountID, jobID string) (bool, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.claim_job_for_execute")
	defer span.End()

	res, err := r.queries.ClaimHubspotSyncJobForExecute(ctx, sqlc.ClaimHubspotSyncJobForExecuteParams{ID: jobID, AccountID: accountID})
	if err != nil {
		return false, tracing.Trace(span, db.MapSQLError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, tracing.Trace(span, db.MapSQLError(err))
	}
	return affected > 0, nil
}

// --- Records ---

func (r *hubspotSyncRepoImpl) UpsertRecord(ctx context.Context, params domain.UpsertHubspotSyncRecordParams) *apierror.APIError {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.upsert_record")
	defer span.End()

	recordID, apiErr := id.GenID(id.HubspotSyncRecordIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if err := r.queries.UpsertHubspotSyncRecord(ctx, sqlc.UpsertHubspotSyncRecordParams{
		ID:          recordID,
		AccountID:   params.AccountID,
		AugnoType:   params.AugnoType,
		AugnoID:     params.AugnoID,
		HubspotType: params.HubspotType,
		HubspotID:   params.HubspotID,
		SyncHash:    db.NullStringPtr(params.SyncHash),
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

// ListRecords pages the account's mappings for one OpenMRP type. Pagination is forward-only: the keyset rides the (account_id, augno_type, augno_id) unique index, and augno_id — not created_at — is the ordering key, so there is no backward branch to mirror.
func (r *hubspotSyncRepoImpl) ListRecords(ctx context.Context, params domain.ListHubspotSyncRecordsParams) (*domain.ListHubspotSyncRecordsResult, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.list_records")
	defer span.End()

	var after sql.NullString
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor"))
		}
		after = sql.NullString{String: cur.ID, Valid: true}
	}

	// Over-fetch by one so BuildPageString can tell whether another page exists.
	rows, err := r.queries.ListHubspotSyncRecords(ctx, sqlc.ListHubspotSyncRecordsParams{
		AccountID: params.AccountID,
		AugnoType: params.AugnoType,
		Cursor:    after,
		Limit:     params.Limit + 1,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	items := make([]*domain.HubspotSyncRecord, len(rows))
	for i, row := range rows {
		items[i] = &domain.HubspotSyncRecord{
			ID:           row.ID,
			AccountID:    row.AccountID,
			AugnoType:    row.AugnoType,
			AugnoID:      row.AugnoID,
			AugnoName:    row.AugnoName,
			HubspotType:  row.HubspotType,
			HubspotID:    row.HubspotID,
			SyncHash:     db.StringFromNullString(row.SyncHash),
			LastSyncedAt: db.TimeFromNullTime(row.LastSyncedAt),
			LastError:    db.StringFromNullString(row.LastError),
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
	}

	page, pageInfo := pagination.BuildPageString(items, params.Limit, nil,
		func(r *domain.HubspotSyncRecord) time.Time { return r.CreatedAt },
		func(r *domain.HubspotSyncRecord) string { return r.AugnoID },
	)
	return &domain.ListHubspotSyncRecordsResult{Items: page, PageInfo: pageInfo}, nil
}

// GetRecord returns the mapping for an OpenMRP entity, or (nil, nil) when none exists.
func (r *hubspotSyncRepoImpl) GetRecord(ctx context.Context, accountID, augnoType, augnoID string) (*domain.HubspotSyncRecord, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.get_record")
	defer span.End()

	row, err := r.queries.GetHubspotSyncRecord(ctx, sqlc.GetHubspotSyncRecordParams{
		AccountID: accountID,
		AugnoType: augnoType,
		AugnoID:   augnoID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return &domain.HubspotSyncRecord{
		ID:           row.ID,
		AccountID:    row.AccountID,
		AugnoType:    row.AugnoType,
		AugnoID:      row.AugnoID,
		HubspotType:  row.HubspotType,
		HubspotID:    row.HubspotID,
		SyncHash:     db.StringFromNullString(row.SyncHash),
		LastSyncedAt: db.TimeFromNullTime(row.LastSyncedAt),
		LastError:    db.StringFromNullString(row.LastError),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

// --- Company reviews ---

func (r *hubspotSyncRepoImpl) CreateReview(ctx context.Context, params domain.CreateHubspotCompanyReviewParams) (*domain.HubspotCompanyReview, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.create_review")
	defer span.End()

	reviewID, apiErr := id.GenID(id.HubspotCompanyReviewIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if err := r.queries.InsertHubspotCompanyReview(ctx, sqlc.InsertHubspotCompanyReviewParams{
		ID:               reviewID,
		JobID:            params.JobID,
		AccountID:        params.AccountID,
		AugnoCustomerID:  params.AugnoCustomerID,
		CustomerName:     params.CustomerName,
		CustomerEmail:    db.NullStringPtr(params.CustomerEmail),
		CustomerUrl:      db.NullStringPtr(params.CustomerURL),
		CandidateMatches: db.NullableRawMessage(params.CandidateMatches),
		Status:           params.Status,
	}); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return r.GetReview(ctx, params.AccountID, reviewID)
}

func (r *hubspotSyncRepoImpl) GetReview(ctx context.Context, accountID, reviewID string) (*domain.HubspotCompanyReview, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.get_review")
	defer span.End()

	row, err := r.queries.GetHubspotCompanyReview(ctx, sqlc.GetHubspotCompanyReviewParams{ID: reviewID, AccountID: accountID})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return mapReview(row), nil
}

// GetReviewsByIDs reads the named reviews in one round trip. Ids that do not exist (or belong to another account) are simply absent from the result — the caller decides what a miss means.
func (r *hubspotSyncRepoImpl) GetReviewsByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.HubspotCompanyReview, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.get_reviews_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetHubspotCompanyReviewsByIDs(ctx, sqlc.GetHubspotCompanyReviewsByIDsParams{AccountID: accountID, Ids: ids})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	reviews := make([]*domain.HubspotCompanyReview, len(rows))
	for i, row := range rows {
		reviews[i] = mapReview(sqlc.HubspotCompanyReview(row))
	}
	return reviews, nil
}

func (r *hubspotSyncRepoImpl) ListReviewsForJob(ctx context.Context, jobID string, status *string) ([]*domain.HubspotCompanyReview, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.list_reviews_for_job")
	defer span.End()

	rows, err := r.queries.ListHubspotCompanyReviewsForJob(ctx, sqlc.ListHubspotCompanyReviewsForJobParams{
		JobID:  jobID,
		Status: db.NullStringPtr(status),
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	reviews := make([]*domain.HubspotCompanyReview, len(rows))
	for i, row := range rows {
		reviews[i] = mapReview(row)
	}
	return reviews, nil
}

func (r *hubspotSyncRepoImpl) CountPendingReviews(ctx context.Context, jobID string) (int64, *apierror.APIError) {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.count_pending_reviews")
	defer span.End()

	count, err := r.queries.CountPendingHubspotCompanyReviews(ctx, jobID)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return count, nil
}

func (r *hubspotSyncRepoImpl) ResolveReview(ctx context.Context, params domain.ResolveHubspotCompanyReviewParams) *apierror.APIError {
	ctx, span := hubspotSyncRepoTracer.Start(ctx, "repository.hubspot_sync.resolve_review")
	defer span.End()

	if _, err := r.queries.ResolveHubspotCompanyReview(ctx, sqlc.ResolveHubspotCompanyReviewParams{
		ID:                params.ID,
		AccountID:         params.AccountID,
		Status:            params.Status,
		Resolution:        db.NullStringPtr(params.Resolution),
		ResolvedHubspotID: db.NullStringPtr(params.ResolvedHubspotID),
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

// --- mappers ---

func mapJob(row sqlc.HubspotSyncJob) *domain.HubspotSyncJob {
	return &domain.HubspotSyncJob{
		ID:             row.ID,
		AccountID:      row.AccountID,
		Status:         row.Status,
		GoLiveCutoffAt: db.TimeFromNullTime(row.GoliveCutoffAt),
		Cursors:        json.RawMessage(row.Cursors),
		Counts:         json.RawMessage(row.Counts),
		LastError:      db.StringFromNullString(row.LastError),
		StartedAt:      db.TimeFromNullTime(row.StartedAt),
		CompletedAt:    db.TimeFromNullTime(row.CompletedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func mapReview(row sqlc.HubspotCompanyReview) *domain.HubspotCompanyReview {
	return &domain.HubspotCompanyReview{
		ID:                row.ID,
		JobID:             row.JobID,
		AccountID:         row.AccountID,
		AugnoCustomerID:   row.AugnoCustomerID,
		CustomerName:      row.CustomerName,
		CustomerEmail:     db.StringFromNullString(row.CustomerEmail),
		CustomerURL:       db.StringFromNullString(row.CustomerUrl),
		CandidateMatches:  json.RawMessage(row.CandidateMatches),
		Status:            row.Status,
		Resolution:        db.StringFromNullString(row.Resolution),
		ResolvedHubspotID: db.StringFromNullString(row.ResolvedHubspotID),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
