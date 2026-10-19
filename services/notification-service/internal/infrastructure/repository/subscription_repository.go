package repository

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

var subscriptionRepoTracer = tracing.GetTracer("notification-service.subscription_repository")

type subscriptionRepoImpl struct {
	db *sqlc.Queries
}

func NewSubscriptionRepo(db *sqlc.Queries) domain.SubscriptionRepo {
	return &subscriptionRepoImpl{db: db}
}

func (r *subscriptionRepoImpl) Get(ctx context.Context, id, accountUserID string) (*domain.Subscription, *apierror.APIError) {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.get")
	defer span.End()
	row, err := r.db.GetSubscription(ctx, sqlc.GetSubscriptionParams{ID: id, AccountUserID: accountUserID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return nil, apierror.NewNotFoundError("Subscription not found.")
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return subscriptionFromRow(row), nil
}

func (r *subscriptionRepoImpl) GetByRecord(ctx context.Context, accountUserID, recordID string) (*domain.Subscription, *apierror.APIError) {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.get_by_record")
	defer span.End()
	row, err := r.db.GetSubscriptionByRecord(ctx, sqlc.GetSubscriptionByRecordParams{AccountUserID: accountUserID, RecordID: recordID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return subscriptionFromRow(row), nil
}

func (r *subscriptionRepoImpl) Insert(ctx context.Context, sub *domain.Subscription) (bool, *apierror.APIError) {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.insert")
	defer span.End()
	res, err := r.db.DB().ExecContext(ctx, insertSubscriptionSQL,
		sub.ID, sub.AccountID, sub.AccountUserID, sub.RecordType, sub.RecordID, sub.Reason, sub.Status)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to read inserted subscription count."))
	}
	return n > 0, nil
}

// insertSubscriptionSQL mirrors the InsertSubscription query; run directly to read the affected-row count.
const insertSubscriptionSQL = `INSERT IGNORE INTO subscription (id, account_id, account_user_id, record_type, record_id, reason, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))`

func (r *subscriptionRepoImpl) SetStatus(ctx context.Context, id, accountUserID, status, reason string) *apierror.APIError {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.set_status")
	defer span.End()
	err := r.db.SetSubscriptionStatus(ctx, sqlc.SetSubscriptionStatusParams{Status: status, Reason: reason, ID: id, AccountUserID: accountUserID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *subscriptionRepoImpl) Subscribers(ctx context.Context, accountID, recordType, recordID string) ([]string, *apierror.APIError) {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.subscribers")
	defer span.End()
	ids, err := r.db.ListRecordSubscribers(ctx, sqlc.ListRecordSubscribersParams{AccountID: accountID, RecordType: recordType, RecordID: recordID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *subscriptionRepoImpl) HasAny(ctx context.Context, accountID, recordType, recordID string) (bool, *apierror.APIError) {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.has_any")
	defer span.End()
	has, err := r.db.RecordHasSubscriptions(ctx, sqlc.RecordHasSubscriptionsParams{AccountID: accountID, RecordType: recordType, RecordID: recordID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return has, nil
}

func (r *subscriptionRepoImpl) List(ctx context.Context, f domain.SubscriptionListFilter) ([]*domain.Subscription, *apierror.APIError) {
	ctx, span := subscriptionRepoTracer.Start(ctx, "repository.subscription.list")
	defer span.End()
	query, args := buildSubscriptionListQuery(f)
	rows, err := r.db.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Subscription
	for rows.Next() {
		var row sqlc.Subscription
		if err := rows.Scan(&row.ID, &row.AccountID, &row.AccountUserID, &row.RecordType, &row.RecordID, &row.Reason,
			&row.Status, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to read subscriptions."))
		}
		out = append(out, subscriptionFromRow(row))
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to read subscriptions."))
	}
	return out, nil
}

// buildSubscriptionListQuery assembles one page of a person's subscriptions, newest first. Only the filters the
// caller set are emitted. A record filter is served by the (account_user_id, record_id) key, a status filter by
// subscription_user_status_created_idx, and no filter by subscription_user_created_idx.
func buildSubscriptionListQuery(f domain.SubscriptionListFilter) (string, []any) {
	var b strings.Builder
	b.WriteString("SELECT id, account_id, account_user_id, record_type, record_id, reason, status, created_at, updated_at" +
		" FROM subscription WHERE account_user_id = ?")
	args := []any{f.AccountUserID}
	in := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		b.WriteString(" AND " + column + " IN (" + strings.TrimSuffix(strings.Repeat("?,", len(values)), ",") + ")")
		for _, v := range values {
			args = append(args, v)
		}
	}
	in("record_id", f.RecordIDs)
	in("record_type", f.RecordTypes)
	in("status", f.Statuses)
	if cursorID := db.NullStringPtr(f.CursorID); cursorID.Valid {
		at := db.NullTimePtr(f.CursorCreatedAt)
		b.WriteString(" AND (created_at < ? OR (created_at = ? AND id < ?))")
		args = append(args, at, at, cursorID)
	}
	b.WriteString(" ORDER BY created_at DESC, id DESC LIMIT ?")
	return b.String(), append(args, f.Limit)
}

func subscriptionFromRow(row sqlc.Subscription) *domain.Subscription {
	return &domain.Subscription{
		ID:            row.ID,
		AccountID:     row.AccountID,
		AccountUserID: row.AccountUserID,
		RecordType:    row.RecordType,
		RecordID:      row.RecordID,
		Reason:        row.Reason,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
