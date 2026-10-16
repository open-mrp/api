package repository

import (
	"context"
	gosql "database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
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

// floatToDecimalString renders a float for a DECIMAL column.
//
// Never use fmt.Sprintf("%v") for this: it switches to scientific notation at large
// and small magnitudes, which MySQL rejects or silently truncates on a DECIMAL column.
// 'f' with -1 precision keeps full round-trippable precision in plain notation.
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

const orderDiscountColumns = `od.id, od.name, od.code, od.percentage, od.value, od.discount_type_code, od.account_id, ` +
	`od.created_at, od.updated_at, od.percent_off, od.amount_off_amount, od.amount_off_currency, od.is_active, od.times_redeemed`

// orderDiscountFromRow reads a stored discount. A row written before the typed columns existed falls back to the
// legacy fraction or amount.
func orderDiscountFromRow(row sqlc.OrderDiscount) *domain.OrderDiscount {
	d := &domain.OrderDiscount{
		ID:               row.ID,
		Name:             row.Name,
		Code:             row.Code,
		Status:           constants.ArchiveStatusFromIsActive(row.IsActive),
		TimesRedeemed:    row.TimesRedeemed,
		Percentage:       floatToDecimalString(row.Percentage),
		Amount:           floatToDecimalString(row.Value),
		DiscountTypeCode: row.DiscountTypeCode,
		AccountID:        row.AccountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if row.DiscountTypeCode == string(constants.OrderDiscountTypeAmount) {
		amount := decimal.NewFromFloat(row.Value).Round(6).String()
		if row.AmountOffAmount.Valid {
			amount = trimDecimal(row.AmountOffAmount.String)
		}
		d.Discount = domain.Discount{Type: constants.DiscountTypeAmountOff, AmountOff: &domain.Money{Amount: amount, Currency: constants.CurrencyUSD}}
		return d
	}
	pct := decimal.NewFromFloat(row.Percentage).Mul(decimal.NewFromInt(100)).Round(6).String()
	if row.PercentOff.Valid {
		pct = trimDecimal(row.PercentOff.String)
	}
	d.Discount = domain.Discount{Type: constants.DiscountTypePercentOff, PercentOff: &pct}
	return d
}

// orderDiscountColumnsFor restates a discount in the stored columns, legacy and typed.
func orderDiscountColumnsFor(d *domain.OrderDiscount) (pct, value float64, typeCode string, percentOff, amount, currency gosql.NullString) {
	switch d.Discount.Type {
	case constants.DiscountTypeAmountOff:
		a := "0"
		if d.Discount.AmountOff != nil {
			a = d.Discount.AmountOff.Amount
		}
		value = stringToFloat(a)
		typeCode = string(constants.OrderDiscountTypeAmount)
		amount = gosql.NullString{String: a, Valid: true}
		currency = gosql.NullString{String: string(constants.CurrencyUSD), Valid: true}
	default:
		p := "0"
		if d.Discount.PercentOff != nil {
			p = *d.Discount.PercentOff
		}
		f, _ := decimal.NewFromString(p)
		pct, _ = f.Div(decimal.NewFromInt(100)).Float64()
		typeCode = string(constants.OrderDiscountTypePercentage)
		percentOff = gosql.NullString{String: p, Valid: true}
	}
	return
}

func (r *orderDiscountRepoImpl) List(ctx context.Context, params domain.ListOrderDiscountsParams) (*domain.ListOrderDiscountsResult, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.list")
	defer span.End()

	cursor, cursorDir, dir, apiErr := decodePriceListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	where := []string{"od.account_id = ?"}
	args := []any{params.AccountID}
	if active := constants.IsActiveFilter(params.Statuses); active != nil {
		where = append(where, "od.is_active = ?")
		args = append(args, *active)
	}
	if params.Query != nil && strings.TrimSpace(*params.Query) != "" {
		like := "%" + db.EscapeLike(strings.TrimSpace(*params.Query)) + "%"
		where = append(where, "(od.name LIKE ? OR od.code LIKE ?)")
		args = append(args, like, like)
	}
	tail, args := keysetTail("od", where, args, cursor, dir, params.Limit+1)
	rows, err := r.queries.DB().QueryContext(ctx, "SELECT "+orderDiscountColumns+" FROM order_discount od"+tail, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var discounts []*domain.OrderDiscount
	for rows.Next() {
		var row sqlc.OrderDiscount
		if err := rows.Scan(&row.ID, &row.Name, &row.Code, &row.Percentage, &row.Value, &row.DiscountTypeCode, &row.AccountID,
			&row.CreatedAt, &row.UpdatedAt, &row.PercentOff, &row.AmountOffAmount, &row.AmountOffCurrency, &row.IsActive, &row.TimesRedeemed); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		discounts = append(discounts, orderDiscountFromRow(row))
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, pageInfo := pagination.BuildPageString(discounts, params.Limit, cursorDir, orderDiscountCreatedAt, orderDiscountID)
	return &domain.ListOrderDiscountsResult{OrderDiscounts: result, PageInfo: pageInfo}, nil
}

func (r *orderDiscountRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetOrderDiscountsByIDs(ctx, sqlc.GetOrderDiscountsByIDsParams{Ids: ids, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.OrderDiscount, len(rows))
	for i, row := range rows {
		out[i] = orderDiscountFromRow(row)
	}
	return out, nil
}

func (r *orderDiscountRepoImpl) Get(ctx context.Context, params domain.GetOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.get")
	defer span.End()

	row, err := r.queries.GetOrderDiscount(ctx, sqlc.GetOrderDiscountParams{ID: params.OrderDiscountID, AccountID: params.AccountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return orderDiscountFromRow(row), nil
}

func (r *orderDiscountRepoImpl) Create(ctx context.Context, d *domain.OrderDiscount) *apierror.APIError {
	pct, value, typeCode, percentOff, amount, currency := orderDiscountColumnsFor(d)
	return db.MapSQLError(r.queries.InsertOrderDiscount(ctx, sqlc.InsertOrderDiscountParams{
		ID:                d.ID,
		Name:              d.Name,
		Code:              d.Code,
		Percentage:        pct,
		Value:             value,
		DiscountTypeCode:  typeCode,
		PercentOff:        percentOff,
		AmountOffAmount:   amount,
		AmountOffCurrency: currency,
		IsActive:          d.Status.IsActive(),
		AccountID:         d.AccountID,
	}))
}

func (r *orderDiscountRepoImpl) Update(ctx context.Context, d *domain.OrderDiscount) *apierror.APIError {
	pct, value, typeCode, percentOff, amount, currency := orderDiscountColumnsFor(d)
	return db.MapSQLError(r.queries.UpdateOrderDiscount(ctx, sqlc.UpdateOrderDiscountParams{
		ID:                d.ID,
		AccountID:         d.AccountID,
		Name:              d.Name,
		Code:              d.Code,
		Percentage:        pct,
		Value:             value,
		DiscountTypeCode:  typeCode,
		PercentOff:        percentOff,
		AmountOffAmount:   amount,
		AmountOffCurrency: currency,
		IsActive:          d.Status.IsActive(),
	}))
}

func (r *orderDiscountRepoImpl) Delete(ctx context.Context, params domain.DeleteOrderDiscountParams) (*domain.OrderDiscount, *apierror.APIError) {
	ctx, span := orderDiscountRepoTracer.Start(ctx, "repository.order_discount.delete")
	defer span.End()

	discount, apiErr := r.Get(ctx, domain.GetOrderDiscountParams(params))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, err := r.queries.DeleteOrderDiscount(ctx, sqlc.DeleteOrderDiscountParams{ID: params.OrderDiscountID, AccountID: params.AccountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Order discount not found."))
	}
	return discount, nil
}

func (r *orderDiscountRepoImpl) AdjustRedemptions(ctx context.Context, id string, delta int32) *apierror.APIError {
	return db.MapSQLError(r.queries.AdjustOrderDiscountRedemptions(ctx, sqlc.AdjustOrderDiscountRedemptionsParams{ID: id, Delta: delta}))
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

	row, err := r.queries.FindOrderDiscountByCode(ctx, sqlc.FindOrderDiscountByCodeParams{Code: code, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return orderDiscountFromRow(row), nil
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
