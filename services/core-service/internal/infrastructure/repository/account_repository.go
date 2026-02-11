package repository

import (
	"context"

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

func (r *accountRepoImpl) Create(ctx context.Context, id, name string, accountTypeCode domain.AccountType, planCode string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.create")
	defer span.End()

	err := r.queries.CreateAccount(ctx, sqlc.CreateAccountParams{
		ID:              id,
		Name:            name,
		AccountTypeCode: string(accountTypeCode),
		PlanCode:        planCode,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRepoImpl) GetPlanCode(ctx context.Context, id string) (string, *apierror.APIError) {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.get_plan_code")
	defer span.End()

	planCode, err := r.queries.GetAccountPlanCode(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return planCode, nil
}

func (r *accountRepoImpl) Delete(ctx context.Context, id string) *apierror.APIError {
	ctx, span := accountRepoTracer.Start(ctx, "repository.account.delete")
	defer span.End()

	err := r.queries.DeleteAccountByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
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

	return &domain.AccountContext{
		AccountID:      row.ID,
		IsSandbox:      isSandbox,
		OwnerAccountID: db.StringFromNullString(row.OwnerAccountID),
		AccountMode:    accountMode,
	}, nil
}
