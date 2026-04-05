package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var salesTargetRepoTracer = tracing.GetTracer("core-service.sales_target_repository")

type salesTargetRepoImpl struct {
	queries *sqlc.Queries
}

func NewSalesTargetRepo(queries *sqlc.Queries) domain.SalesTargetRepo {
	return &salesTargetRepoImpl{queries: queries}
}

func (r *salesTargetRepoImpl) List(ctx context.Context, params domain.ListSalesTargetsParams) (*domain.ListSalesTargetsResult, *apierror.APIError) {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.list")
	defer span.End()

	var search interface{}
	if params.Query != nil && *params.Query != "" {
		search = *params.Query
	}

	rows, err := r.queries.ListSalesTargets(ctx, sqlc.ListSalesTargetsParams{
		SalesRepID: params.SalesRepID,
		AccountID:  params.AccountID,
		Search:     search,
		Limit:      params.Limit,
		Offset:     params.Offset,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	count, err := r.queries.CountSalesTargets(ctx, sqlc.CountSalesTargetsParams{
		SalesRepID: params.SalesRepID,
		AccountID:  params.AccountID,
		Search:     search,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	targets := make([]domain.SalesTarget, len(rows))
	for i, row := range rows {
		targets[i] = domain.SalesTarget{
			ID:           row.ID,
			StartDate:    row.StartDate,
			EndDate:      row.EndDate,
			SalesRepID:   row.SalesRepID,
			AccountID:    row.AccountID,
			AmountID:     row.AmountID,
			AmountValue:  row.AmountValue,
			AmountUnitID: row.AmountUnitID,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
	}

	return &domain.ListSalesTargetsResult{
		SalesTargets: targets,
		Total:        count,
	}, nil
}

func (r *salesTargetRepoImpl) Get(ctx context.Context, targetID string) (*domain.SalesTarget, *apierror.APIError) {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.get")
	defer span.End()

	row, err := r.queries.GetSalesTarget(ctx, targetID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.SalesTarget{
		ID:           row.ID,
		StartDate:    row.StartDate,
		EndDate:      row.EndDate,
		SalesRepID:   row.SalesRepID,
		AccountID:    row.AccountID,
		AmountID:     row.AmountID,
		AmountValue:  row.AmountValue,
		AmountUnitID: row.AmountUnitID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *salesTargetRepoImpl) Exists(ctx context.Context, targetID string) (bool, *apierror.APIError) {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.exists")
	defer span.End()

	exists, err := r.queries.SalesTargetExists(ctx, targetID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *salesTargetRepoImpl) IsInAccount(ctx context.Context, targetID, accountID string) (bool, *apierror.APIError) {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.is_in_account")
	defer span.End()

	isInAccount, err := r.queries.SalesTargetIsInAccount(ctx, sqlc.SalesTargetIsInAccountParams{
		ID:        targetID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return isInAccount, nil
}

func (r *salesTargetRepoImpl) SalesRepExistsInAccount(ctx context.Context, salesRepID, accountID string) (bool, *apierror.APIError) {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.sales_rep_exists_in_account")
	defer span.End()

	exists, err := r.queries.SalesRepExistsInAccount(ctx, sqlc.SalesRepExistsInAccountParams{
		ID:        salesRepID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *salesTargetRepoImpl) Create(ctx context.Context, id string, params domain.CreateSalesTargetParams, amountID string) *apierror.APIError {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.create")
	defer span.End()

	err := r.queries.InsertSalesTarget(ctx, sqlc.InsertSalesTargetParams{
		ID:         id,
		StartDate:  params.StartDate,
		EndDate:    params.EndDate,
		SalesRepID: params.SalesRepID,
		AccountID:  params.AccountID,
		AmountID:   amountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesTargetRepoImpl) Update(ctx context.Context, params domain.UpsertSalesTargetParams) *apierror.APIError {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.update")
	defer span.End()

	err := r.queries.UpdateSalesTarget(ctx, sqlc.UpdateSalesTargetParams{
		ID:        params.TargetID,
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesTargetRepoImpl) InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.insert_quantity")
	defer span.End()

	err := r.queries.InsertQuantity(ctx, sqlc.InsertQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *salesTargetRepoImpl) UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := salesTargetRepoTracer.Start(ctx, "repository.sales_target.update_quantity")
	defer span.End()

	_, err := r.queries.UpdateQuantity(ctx, sqlc.UpdateQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
