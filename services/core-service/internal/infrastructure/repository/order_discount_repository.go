package repository

import (
	"context"
	gosql "database/sql"
	"strconv"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/safeconv"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var orderDiscountRepoTracer = tracing.GetTracer("core-service.order_discount_repository")

type orderDiscountRepoImpl struct {
	queries *sqlc.Queries
}

func NewOrderDiscountRepo(queries *sqlc.Queries) domain.OrderDiscountRepo {
	return &orderDiscountRepoImpl{queries: queries}
}

func orderDiscountCreatedAt(d *domain.OrderDiscount) time.Time { return d.CreatedAt }
func orderDiscountID(d *domain.OrderDiscount) string           { return d.ID }

func floatToDecimalString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func stringToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func toNullFloat64(s *string) gosql.NullFloat64 {
	if s == nil {
		return gosql.NullFloat64{}
	}
	f, _ := strconv.ParseFloat(*s, 64)
	return gosql.NullFloat64{Float64: f, Valid: true}
}

func buildOrderDiscountSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func mapOrderDiscountForwardRow(row sqlc.ListOrderDiscountsForwardRow) *domain.OrderDiscount {
	return &domain.OrderDiscount{
		ID:               row.ID,
		Name:             row.Name,
		Code:             row.Code,
		Percentage:       floatToDecimalString(row.Percentage),
		Amount:           floatToDecimalString(row.Value),
		DiscountTypeCode: row.DiscountTypeCode,
		AccountID:        row.AccountID,
		OrderCount:       safeconv.Int64ToInt32(row.OrderCount),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapOrderDiscountBackwardRow(row sqlc.ListOrderDiscountsBackwardRow) *domain.OrderDiscount {
	return &domain.OrderDiscount{
		ID:               row.ID,
		Name:             row.Name,
		Code:             row.Code,
		Percentage:       floatToDecimalString(row.Percentage),
		Amount:           floatToDecimalString(row.Value),
		DiscountTypeCode: row.DiscountTypeCode,
		AccountID:        row.AccountID,
		OrderCount:       safeconv.Int64ToInt32(row.OrderCount),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapGetOrderDiscountRow(row sqlc.GetOrderDiscountRow) *domain.OrderDiscount {
	return &domain.OrderDiscount{
		ID:               row.ID,
		Name:             row.Name,
		Code:             row.Code,
		Percentage:       floatToDecimalString(row.Percentage),
		Amount:           floatToDecimalString(row.Value),
		DiscountTypeCode: row.DiscountTypeCode,
		AccountID:        row.AccountID,
		OrderCount:       safeconv.Int64ToInt32(row.OrderCount),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapFindOrderDiscountByCodeRow(row sqlc.FindOrderDiscountByCodeRow) *domain.OrderDiscount {
	return &domain.OrderDiscount{
		ID:               row.ID,
		Name:             row.Name,
		Code:             row.Code,
		Percentage:       floatToDecimalString(row.Percentage),
		Amount:           floatToDecimalString(row.Value),
		DiscountTypeCode: row.DiscountTypeCode,
		AccountID:        row.AccountID,
		OrderCount:       safeconv.Int64ToInt32(row.OrderCount),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (r *orderDiscountRepoImpl) List(ctx context.Context, params domain.ListOrderDiscountsParams) (*domain.ListOrderDiscountsResult, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.list")
	defer span.End()

	searchQuery := buildOrderDiscountSearchParams(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListOrderDiscountsBackward(ctx, sqlc.ListOrderDiscountsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			discounts := make([]*domain.OrderDiscount, len(rows))
			for i, row := range rows {
				discounts[i] = mapOrderDiscountBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, orderDiscountCreatedAt, orderDiscountID)
			return &domain.ListOrderDiscountsResult{OrderDiscounts: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListOrderDiscountsForward(ctx, sqlc.ListOrderDiscountsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		discounts := make([]*domain.OrderDiscount, len(rows))
		for i, row := range rows {
			discounts[i] = mapOrderDiscountForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, orderDiscountCreatedAt, orderDiscountID)
		return &domain.ListOrderDiscountsResult{OrderDiscounts: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListOrderDiscountsForward(ctx, sqlc.ListOrderDiscountsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	discounts := make([]*domain.OrderDiscount, len(rows))
	for i, row := range rows {
		discounts[i] = mapOrderDiscountForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, orderDiscountCreatedAt, orderDiscountID)
	return &domain.ListOrderDiscountsResult{OrderDiscounts: result, PageInfo: pageInfo}, nil
}

func (r *orderDiscountRepoImpl) Get(ctx context.Context, params domain.GetOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.get")
	defer span.End()

	row, err := r.queries.GetOrderDiscount(ctx, sqlc.GetOrderDiscountParams{
		ID:        params.OrderDiscountID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetOrderDiscountRow(row), nil
}

func (r *orderDiscountRepoImpl) Create(ctx context.Context, id string, params domain.CreateOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.create")
	defer span.End()

	var percentage float64
	if params.Percentage != nil {
		percentage = stringToFloat(*params.Percentage)
	}
	var amount float64
	if params.Amount != nil {
		amount = stringToFloat(*params.Amount)
	}

	err := r.queries.InsertOrderDiscount(ctx, sqlc.InsertOrderDiscountParams{
		ID:               id,
		Name:             params.Name,
		Code:             params.Code,
		Percentage:       percentage,
		Value:            amount,
		DiscountTypeCode: params.DiscountType,
		AccountID:        params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetOrderDiscountParams{AccountID: params.AccountID, OrderDiscountID: id})
}

func (r *orderDiscountRepoImpl) Update(ctx context.Context, params domain.UpdateOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.update")
	defer span.End()

	result, err := r.queries.UpdateOrderDiscount(ctx, sqlc.UpdateOrderDiscountParams{
		ID:               params.OrderDiscountID,
		AccountID:        params.AccountID,
		Name:             toNullString(params.Name),
		Code:             toNullString(params.Code),
		Percentage:       toNullFloat64(params.Percentage),
		Value:            toNullFloat64(params.Amount),
		DiscountTypeCode: toNullString(params.DiscountType),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Order discount not found."))
	}

	return r.Get(ctx, domain.GetOrderDiscountParams{AccountID: params.AccountID, OrderDiscountID: params.OrderDiscountID})
}

func (r *orderDiscountRepoImpl) Delete(ctx context.Context, params domain.DeleteOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.delete")
	defer span.End()

	// Fetch the order discount before deleting so we can return it.
	discount, apiErr := r.Get(ctx, domain.GetOrderDiscountParams(params))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, err := r.queries.DeleteOrderDiscount(ctx, sqlc.DeleteOrderDiscountParams{
		ID:        params.OrderDiscountID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Order discount not found."))
	}

	return discount, nil
}

func (r *orderDiscountRepoImpl) ExistsByCode(ctx context.Context, accountID, code string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.exists_by_code")
	defer span.End()

	count, err := r.queries.CountOrderDiscountsByCode(ctx, sqlc.CountOrderDiscountsByCodeParams{
		AccountID: accountID,
		Code:      code,
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *orderDiscountRepoImpl) FindByCode(ctx context.Context, accountID, code string) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.find_by_code")
	defer span.End()

	row, err := r.queries.FindOrderDiscountByCode(ctx, sqlc.FindOrderDiscountByCodeParams{
		Code:      code,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapFindOrderDiscountByCodeRow(row), nil
}

func (r *orderDiscountRepoImpl) CheckDuplicateUsage(ctx context.Context, accountID, buyerAccountID, orderDiscountID string, excludeOrderID *string) (bool, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.check_duplicate_usage")
	defer span.End()

	count, err := r.queries.CheckOrderDiscountDuplicateUsage(ctx, sqlc.CheckOrderDiscountDuplicateUsageParams{
		OrderDiscountID: gosql.NullString{String: orderDiscountID, Valid: true},
		BuyerAccountID:  buyerAccountID,
		SellerAccountID: accountID,
		OwnerAccountID:  accountID,
		ExcludeOrderID:  toNullString(excludeOrderID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
