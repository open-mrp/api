package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountRepoTracer = tracing.GetTracer("core-service.account_repository")

type accountRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountRepo(queries *sqlc.Queries) domain.AccountRepo {
	return &accountRepoImpl{queries: queries}
}

func (r *accountRepoImpl) Create(ctx context.Context, id, name string, accountTypeCode domain.AccountType, planCode constants.PlanCode) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.create")
	defer span.End()

	err := r.queries.CreateAccount(ctx, sqlc.CreateAccountParams{
		ID:              id,
		Name:            name,
		AccountTypeCode: string(accountTypeCode),
		PlanCode:        string(planCode),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetPlanCode(ctx context.Context, id string) (constants.PlanCode, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_plan_code")
	defer span.End()

	planCodeStr, err := r.queries.GetAccountPlanCode(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	planCode := constants.PlanCode(planCodeStr)
	if !planCode.IsValid() {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Invalid plan code: %s", planCodeStr)))
	}

	return planCode, nil
}

func (r *accountRepoImpl) Delete(ctx context.Context, id string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.delete")
	defer span.End()

	result, err := r.queries.DeleteAccountByIDIfSandbox(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected"))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Account is not a sandbox account or does not exist."))
	}

	return nil
}

func (r *accountRepoImpl) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_account_context")
	defer span.End()

	row, err := r.queries.GetAccountContext(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	isSandbox := row.AccountTypeCode == string(constants.AccountTypeCodeSandbox)
	accountMode := constants.AccountModeProduction
	if isSandbox {
		accountMode = constants.AccountModeSandbox
	}

	var subscriptionStatus *string
	if row.SubscriptionStatus != nil {
		if s, ok := row.SubscriptionStatus.(string); ok && s != "" {
			subscriptionStatus = &s
		}
	}

	return &domain.AccountContext{
		AccountID:          row.ID,
		IsSandbox:          isSandbox,
		OwnerAccountID:     db.StringFromNullString(row.OwnerAccountID),
		AccountMode:        accountMode,
		SubscriptionStatus: subscriptionStatus,
	}, nil
}

func (r *accountRepoImpl) GetPlanTypeIDByCode(ctx context.Context, planCode string) (string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_plan_type_id_by_code")
	defer span.End()

	typeID, err := r.queries.GetAccountPlanTypeIDByCode(ctx, planCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return typeID, nil
}

func (r *accountRepoImpl) UpdateSubscription(ctx context.Context, accountID string, status *string, planCode string, accountPlanID *string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.update_subscription")
	defer span.End()

	err := r.queries.UpdateAccountSubscription(ctx, sqlc.UpdateAccountSubscriptionParams{
		ID:                           accountID,
		SubscriptionStatus:           db.NullStringPtr(status),
		PlanCode:                     planCode,
		AccountPlanID:                db.NullStringPtr(accountPlanID),
		InternalStripeSubscriptionID: db.NullStringPtr(stripeSubID),
		SubscriptionCurrentPeriodEnd: db.NullTimePtr(periodEnd),
		InternalStripeCustomerID:     db.NullStringPtr(stripeCustomerID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) ClearStripeCustomer(ctx context.Context, accountID string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.clear_stripe_customer")
	defer span.End()

	err := r.queries.ClearAccountStripeCustomer(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetSandboxLimit(ctx context.Context, accountID string) (*int32, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_sandbox_limit")
	defer span.End()

	result, err := r.queries.GetSandboxLimitByAccountID(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	val := result.Int32
	return &val, nil
}

func (r *accountRepoImpl) GetSeatLimitByPlanCode(ctx context.Context, planCode string) (*int32, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_seat_limit_by_plan_code")
	defer span.End()

	result, err := r.queries.GetSeatLimitByPlanCode(ctx, planCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !result.Valid {
		return nil, nil
	}

	val := result.Int32
	return &val, nil
}

func (r *accountRepoImpl) CountNonSandboxByPlanCode(ctx context.Context, planCode string) (int64, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.count_non_sandbox_by_plan_code")
	defer span.End()

	count, err := r.queries.CountNonSandboxAccountsByPlanCode(ctx, planCode)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *accountRepoImpl) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_by_stripe_customer_id")
	defer span.End()

	row, err := r.queries.GetAccountByStripeCustomerID(ctx, sql.NullString{String: stripeCustomerID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", "", tracing.Trace(span, apiErr)
	}

	return row.ID, row.PlanCode, nil
}
