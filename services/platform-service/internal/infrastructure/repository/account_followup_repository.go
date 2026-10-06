package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var accountFollowupRepoTracer = tracing.GetTracer("platform-service.account_followup_repository")

type accountFollowupRepoImpl struct {
	db *sqlc.Queries
}

func NewAccountFollowupRepo(db *sqlc.Queries) domain.AccountFollowupRepo {
	return &accountFollowupRepoImpl{db: db}
}

func (r *accountFollowupRepoImpl) Create(ctx context.Context, f *domain.AccountFollowup) *apierror.APIError {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.create")
	defer span.End()

	err := r.db.CreateAccountFollowup(ctx, sqlc.CreateAccountFollowupParams{
		ID:               f.ID,
		AccountID:        f.AccountID,
		SandboxAccountID: db.NullStringPtr(f.SandboxAccountID),
		UserID:           f.UserID,
		AccountName:      f.AccountName,
		RegistrantName:   f.RegistrantName,
		RegistrantEmail:  f.RegistrantEmail,
		RegisteredAt:     f.RegisteredAt,
		Status:           string(f.Status),
		ScheduledFor:     f.ScheduledFor,
	})
	// account_id is unique: a duplicate is a redelivered schedule command for a follow-up that already exists.
	if db.IsDuplicateEntry(err) {
		return nil
	}
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create account follow-up."))
	}
	return nil
}

func (r *accountFollowupRepoImpl) FindByID(ctx context.Context, id string) (*domain.AccountFollowup, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.find_by_id")
	defer span.End()

	row, err := r.db.FindAccountFollowupByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account follow-up not found."))
	}
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to find account follow-up."))
	}

	return mapAccountFollowup(row), nil
}

func (r *accountFollowupRepoImpl) FindByReviewTokenHash(ctx context.Context, hash []byte) (*domain.AccountFollowup, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.find_by_review_token_hash")
	defer span.End()

	row, err := r.db.FindAccountFollowupByReviewTokenHash(ctx, db.NullString(string(hash)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account follow-up not found."))
	}
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to find account follow-up."))
	}
	return mapAccountFollowup(row), nil
}

func (r *accountFollowupRepoImpl) Approve(ctx context.Context, id, subject, body string, now time.Time) (bool, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.approve")
	defer span.End()

	n, err := r.db.ApproveAccountFollowup(ctx, sqlc.ApproveAccountFollowupParams{
		FinalSubject: db.NullString(subject),
		FinalBody:    db.NullString(body),
		Now:          db.NullTime(now),
		ID:           id,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to approve account follow-up."))
	}
	return n == 1, nil
}

func (r *accountFollowupRepoImpl) SkipReview(ctx context.Context, id string, now time.Time) (bool, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.skip_review")
	defer span.End()

	n, err := r.db.SkipReviewedAccountFollowup(ctx, sqlc.SkipReviewedAccountFollowupParams{
		SkipReason: db.NullString(string(domain.AccountFollowupSkipReasonReviewer)),
		Now:        db.NullTime(now),
		ID:         id,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to skip account follow-up."))
	}
	return n == 1, nil
}

func mapAccountFollowup(row sqlc.AccountFollowup) *domain.AccountFollowup {
	f := &domain.AccountFollowup{
		ID:                   row.ID,
		AccountID:            row.AccountID,
		SandboxAccountID:     db.StringFromNullString(row.SandboxAccountID),
		UserID:               row.UserID,
		AccountName:          row.AccountName,
		RegistrantName:       row.RegistrantName,
		RegistrantEmail:      row.RegistrantEmail,
		RegisteredAt:         row.RegisteredAt,
		Status:               domain.AccountFollowupStatus(row.Status),
		ScheduledFor:         row.ScheduledFor,
		Attempts:             int(row.Attempts),
		Engagement:           domain.AccountFollowupEngagement(row.Engagement.String),
		InternalSummary:      row.InternalSummary.String,
		DraftSubject:         row.DraftSubject.String,
		DraftBody:            row.DraftBody.String,
		ReviewTokenExpiresAt: db.TimeFromNullTime(row.ReviewTokenExpiresAt),
		FinalSubject:         row.FinalSubject.String,
		FinalBody:            row.FinalBody.String,
		ReviewedAt:           db.TimeFromNullTime(row.ReviewedAt),
	}
	// A summary that no longer decodes is left out rather than failing the read: the follow-up is still reviewable without it.
	if len(row.ActivitySummary) > 0 {
		var activity domain.AccountActivity
		if err := json.Unmarshal(row.ActivitySummary, &activity); err == nil {
			f.Activity = &activity
		}
	}
	return f
}

func (r *accountFollowupRepoImpl) ListDueIDs(ctx context.Context, now, staleBefore time.Time, limit int32) ([]string, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.list_due_ids")
	defer span.End()

	ids, err := r.db.ListDueAccountFollowupIDs(ctx, sqlc.ListDueAccountFollowupIDsParams{
		Now:         now,
		StaleBefore: staleBefore,
		Limit:       limit,
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to list due account follow-ups."))
	}
	return ids, nil
}

func (r *accountFollowupRepoImpl) ClaimForDraft(ctx context.Context, id string, staleBefore time.Time) (bool, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.claim_for_draft")
	defer span.End()

	n, err := r.db.ClaimAccountFollowupForDraft(ctx, sqlc.ClaimAccountFollowupForDraftParams{
		ID:          id,
		StaleBefore: staleBefore,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to claim account follow-up."))
	}
	return n == 1, nil
}

func (r *accountFollowupRepoImpl) Skip(ctx context.Context, id string, reason domain.AccountFollowupSkipReason) *apierror.APIError {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.skip")
	defer span.End()

	if err := r.db.SkipAccountFollowup(ctx, sqlc.SkipAccountFollowupParams{
		SkipReason: db.NullString(string(reason)),
		ID:         id,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to skip account follow-up."))
	}
	return nil
}

func (r *accountFollowupRepoImpl) Reschedule(ctx context.Context, id string, at time.Time, lastError string) *apierror.APIError {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.reschedule")
	defer span.End()

	if err := r.db.RescheduleAccountFollowup(ctx, sqlc.RescheduleAccountFollowupParams{
		ScheduledFor: at,
		LastError:    db.NullString(lastError),
		ID:           id,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to reschedule account follow-up."))
	}
	return nil
}

func (r *accountFollowupRepoImpl) Fail(ctx context.Context, id string, lastError string) *apierror.APIError {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.fail")
	defer span.End()

	if err := r.db.FailAccountFollowup(ctx, sqlc.FailAccountFollowupParams{
		LastError: db.NullString(lastError),
		ID:        id,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to fail account follow-up."))
	}
	return nil
}

func (r *accountFollowupRepoImpl) SaveDraft(ctx context.Context, id string, d *domain.AccountFollowupDraft) (bool, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.save_draft")
	defer span.End()

	activity, err := json.Marshal(d.Activity)
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to encode account activity."))
	}

	n, err := r.db.SaveAccountFollowupDraft(ctx, sqlc.SaveAccountFollowupDraftParams{
		ActivitySummary:      db.NullableRawMessage(activity),
		LlmModel:             db.NullString(d.Model),
		PromptVersion:        db.NullString(d.PromptVersion),
		Engagement:           db.NullString(string(d.Engagement)),
		InternalSummary:      db.NullString(d.InternalSummary),
		DraftSubject:         db.NullString(d.Subject),
		DraftBody:            db.NullString(d.Body),
		ReviewTokenHash:      db.NullString(string(d.ReviewTokenHash)),
		ReviewTokenExpiresAt: db.NullTime(d.ReviewTokenExpiresAt),
		DraftedAt:            db.NullTime(d.DraftedAt),
		ID:                   id,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to save account follow-up draft."))
	}
	return n == 1, nil
}

func (r *accountFollowupRepoImpl) ListRequests(ctx context.Context, accountID string, sandboxAccountID *string, from, to time.Time, limit int32) ([]domain.AccountRequest, *apierror.APIError) {
	ctx, span := accountFollowupRepoTracer.Start(ctx, "repository.account_followup.list_requests")
	defer span.End()

	// Without a sandbox the second IN slot repeats the account, which matches nothing new.
	sandbox := accountID
	if sandboxAccountID != nil && *sandboxAccountID != "" {
		sandbox = *sandboxAccountID
	}

	rows, err := r.db.ListAccountRequestActivity(ctx, sqlc.ListAccountRequestActivityParams{
		AccountID:        db.NullString(accountID),
		SandboxAccountID: db.NullString(sandbox),
		WindowStart:      from,
		WindowEnd:        to,
		Limit:            limit,
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to list account requests."))
	}

	requests := make([]domain.AccountRequest, len(rows))
	for i, row := range rows {
		requests[i] = domain.AccountRequest{
			TargetAccountID: row.TargetAccountID.String,
			Method:          row.Method,
			NormalizedRoute: row.NormalizedRoute,
			StatusCode:      int(row.StatusCode),
			OccurredAt:      row.OccurredAt,
		}
	}
	return requests, nil
}
