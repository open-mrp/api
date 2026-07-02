package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
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
		DryRun:         params.DryRun,
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

// GetRecord returns the mapping for an Augno entity, or (nil, nil) when none exists.
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
		DryRun:         row.DryRun,
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
		CandidateMatches:  json.RawMessage(row.CandidateMatches),
		Status:            row.Status,
		Resolution:        db.StringFromNullString(row.Resolution),
		ResolvedHubspotID: db.StringFromNullString(row.ResolvedHubspotID),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
