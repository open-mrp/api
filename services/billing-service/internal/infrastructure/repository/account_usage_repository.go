package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountUsageRepoTracer = tracing.GetTracer("billing-service.account_usage_repository")

type accountUsageRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountUsageRepo(queries *sqlc.Queries) domain.AccountUsageRepo {
	return &accountUsageRepoImpl{queries: queries}
}

func (r *accountUsageRepoImpl) GetLimitsByAccountID(ctx context.Context, accountID string) ([]domain.PlanLimit, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_limits_by_account_id")
	defer span.End()

	rows, err := r.queries.GetLimitsByAccountID(ctx, accountID)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	limits := make([]domain.PlanLimit, len(rows))
	for i, row := range rows {
		var value *int
		if row.Value.Valid {
			v := int(row.Value.Int32)
			value = &v
		}
		limits[i] = domain.PlanLimit{
			Key:   row.Key,
			Value: value,
		}
	}

	return limits, nil
}

func (r *accountUsageRepoImpl) CountUsersByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.count_users_by_account_id")
	defer span.End()

	cnt, err := r.queries.CountUsersByAccountID(ctx, accountID)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return int(cnt), nil
}

func (r *accountUsageRepoImpl) CountSandboxesByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.count_sandboxes_by_account_id")
	defer span.End()

	cnt, err := r.queries.CountSandboxesByAccountID(ctx, accountID)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return int(cnt), nil
}

func (r *accountUsageRepoImpl) CountInvoicesByAccountID(ctx context.Context, accountID string, periodStart time.Time) (int, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.count_invoices_by_account_id")
	defer span.End()

	cnt, err := r.queries.CountInvoicesByAccountIDInPeriod(ctx, sqlc.CountInvoicesByAccountIDInPeriodParams{
		AccountID: accountID,
		CreatedAt: periodStart,
	})
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return int(cnt), nil
}

func (r *accountUsageRepoImpl) CountBatchesByAccountID(ctx context.Context, accountID string, periodStart time.Time) (int, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.count_batches_by_account_id")
	defer span.End()

	cnt, err := r.queries.CountBatchesByAccountIDInPeriod(ctx, sqlc.CountBatchesByAccountIDInPeriodParams{
		AccountID: accountID,
		CreatedAt: periodStart,
	})
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return int(cnt), nil
}

func (r *accountUsageRepoImpl) GetAccountSubscriptionInfo(ctx context.Context, accountID string) (*domain.AccountSubscriptionInfo, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_account_subscription_info")
	defer span.End()

	row, err := r.queries.GetAccountSubscriptionInfo(ctx, accountID)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	info := &domain.AccountSubscriptionInfo{}
	if row.SubscriptionStatus.Valid {
		info.SubscriptionStatus = &row.SubscriptionStatus.String
	}
	if row.SubscriptionCurrentPeriodEnd.Valid {
		t := row.SubscriptionCurrentPeriodEnd.Time.UTC()
		info.SubscriptionCurrentPeriodEnd = &t
	}
	if row.InternalStripeSubscriptionID.Valid {
		info.StripeSubscriptionID = &row.InternalStripeSubscriptionID.String
	}
	if row.StripeBillingProfileID.Valid {
		info.BillingProfileID = &row.StripeBillingProfileID.String
	}
	if row.StripeBillingCadenceID.Valid {
		info.BillingCadenceID = &row.StripeBillingCadenceID.String
	}
	if row.StripePricingPlanSubscriptionID.Valid {
		info.PricingPlanSubscriptionID = &row.StripePricingPlanSubscriptionID.String
	}
	if row.ServicingStatus.Valid {
		info.ServicingStatus = &row.ServicingStatus.String
	}
	if row.CollectionStatus.Valid {
		info.CollectionStatus = &row.CollectionStatus.String
	}

	return info, nil
}

func (r *accountUsageRepoImpl) GetStripeCustomerIDByAccountID(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_stripe_customer_id_by_account_id")
	defer span.End()

	row, err := r.queries.GetStripeCustomerIDByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	if row.Valid {
		return &row.String, nil
	}
	return nil, nil
}

func (r *accountUsageRepoImpl) GetAccountStripePricingPlanID(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_account_stripe_pricing_plan_id")
	defer span.End()

	row, err := r.queries.GetAccountStripePricingPlanID(ctx, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	if row.Valid {
		return &row.String, nil
	}
	return nil, nil
}

func (r *accountUsageRepoImpl) GetAccountNameAndPlanCode(ctx context.Context, accountID string) (string, string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_account_name_and_plan_code")
	defer span.End()

	row, err := r.queries.GetAccountNameAndPlanCode(ctx, accountID)
	if err != nil {
		return "", "", tracing.Trace(span, db.MapSQLError(err))
	}

	return row.Name, row.PlanCode, nil
}

func (r *accountUsageRepoImpl) GetUserEmailByID(ctx context.Context, userID string) (string, *string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_user_email_by_id")
	defer span.End()

	row, err := r.queries.GetUserEmailByID(ctx, userID)
	if err != nil {
		return "", nil, tracing.Trace(span, db.MapSQLError(err))
	}

	var email string
	if row.Email.Valid {
		email = row.Email.String
	}

	var displayName *string
	if row.DisplayName.Valid {
		displayName = &row.DisplayName.String
	}

	return email, displayName, nil
}

func (r *accountUsageRepoImpl) GetAdminEmailByAccountID(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.get_admin_email_by_account_id")
	defer span.End()

	row, err := r.queries.GetAdminEmailByAccountID(ctx, accountID)
	if err != nil {
		return "", tracing.Trace(span, db.MapSQLError(err))
	}

	if row.Valid {
		return row.String, nil
	}
	return "", nil
}

func (r *accountUsageRepoImpl) UpdateStripeCustomerIDByAccountID(ctx context.Context, stripeCustomerID, accountID string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, accountUsageRepoTracer, "repository.account_usage.update_stripe_customer_id_by_account_id")
	defer span.End()

	err := r.queries.UpdateStripeCustomerIDByAccountID(ctx, sqlc.UpdateStripeCustomerIDByAccountIDParams{
		InternalStripeCustomerID: sql.NullString{String: stripeCustomerID, Valid: true},
		AccountID:                accountID,
	})
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

// Ensure compile-time interface compliance.
var _ domain.AccountUsageRepo = (*accountUsageRepoImpl)(nil)
