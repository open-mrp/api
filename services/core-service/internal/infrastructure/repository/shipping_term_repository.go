package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var shippingTermRepoTracer = tracing.GetTracer("core-service.shipping_term_repository")

type shippingTermRepoImpl struct {
	queries *sqlc.Queries
}

func NewShippingTermRepo(queries *sqlc.Queries) domain.ShippingTermRepo {
	return &shippingTermRepoImpl{queries: queries}
}

func shippingTermCreatedAt(st *domain.ShippingTerm) time.Time { return st.CreatedAt }
func shippingTermID(st *domain.ShippingTerm) string           { return st.ID }

func buildShippingTermSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

// shippingTermTypeFromBooleans maps DB booleans to the domain ShippingTermType.
func shippingTermTypeFromBooleans(isFreightExempt, isCarrierRate bool) constants.ShippingTermType {
	if isFreightExempt {
		return constants.ShippingTermTypeFree
	}
	if isCarrierRate {
		return constants.ShippingTermTypeCarrierRate
	}
	return constants.ShippingTermTypeFlatRate
}

// shippingTermTypeToBooleans maps a ShippingTermType to DB booleans (isFreightExempt, isCarrierRate).
func shippingTermTypeToBooleans(t constants.ShippingTermType) (bool, bool) {
	switch t {
	case constants.ShippingTermTypeFree:
		return true, false
	case constants.ShippingTermTypeCarrierRate:
		return false, true
	default: // flat_rate
		return false, false
	}
}

type qtyJoinCols struct {
	QtyID                 gosql.NullString
	Value                 gosql.NullString
	UnitID                gosql.NullString
	QtyCreatedAt          gosql.NullTime
	QtyUpdatedAt          gosql.NullTime
	UnitName              gosql.NullString
	UnitAbbreviation      gosql.NullString
	UnitType              gosql.NullString
	UnitRatioNumerator    gosql.NullString
	UnitRatioDenominator  gosql.NullString
	UnitOffsetNumerator   gosql.NullString
	UnitOffsetDenominator gosql.NullString
	UnitIsBaseUnit        gosql.NullBool
	UnitAccountID         gosql.NullString
	UnitCreatedAt         gosql.NullTime
	UnitUpdatedAt         gosql.NullTime
}

func nullSQLString(ns gosql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func mapShippingTermQuantityFromCols(c qtyJoinCols) *domain.Quantity {
	if !c.QtyID.Valid {
		return nil
	}
	q := &domain.Quantity{
		ID:               c.QtyID.String,
		Value:            c.Value.String,
		UnitID:           c.UnitID.String,
		UnitAbbreviation: c.UnitAbbreviation.String,
		UnitType:         c.UnitType.String,
	}
	if c.QtyCreatedAt.Valid {
		q.CreatedAt = c.QtyCreatedAt.Time
	}
	if c.QtyUpdatedAt.Valid {
		q.UpdatedAt = c.QtyUpdatedAt.Time
	}
	if c.UnitName.Valid {
		q.UnitName = c.UnitName.String
	}
	if c.UnitID.Valid && (c.UnitName.Valid || c.UnitCreatedAt.Valid) {
		q.EmbeddedUnit = embeddedUnitFromQtyJoinCols(c)
	}
	return q
}

func embeddedUnitFromQtyJoinCols(c qtyJoinCols) *domain.Unit {
	u := &domain.Unit{
		ID:                c.UnitID.String,
		Name:              nullSQLString(c.UnitName),
		Abbreviation:      nullSQLString(c.UnitAbbreviation),
		UnitDimensionCode: nullSQLString(c.UnitType),
		RatioNumerator:    nullSQLString(c.UnitRatioNumerator),
		RatioDenominator:  nullSQLString(c.UnitRatioDenominator),
		OffsetNumerator:   nullSQLString(c.UnitOffsetNumerator),
		OffsetDenominator: nullSQLString(c.UnitOffsetDenominator),
	}
	if c.UnitIsBaseUnit.Valid {
		u.IsBaseUnit = c.UnitIsBaseUnit.Bool
	}
	if c.UnitAccountID.Valid {
		a := c.UnitAccountID.String
		u.AccountID = &a
	}
	if c.UnitCreatedAt.Valid {
		u.CreatedAt = c.UnitCreatedAt.Time
	}
	if c.UnitUpdatedAt.Valid {
		u.UpdatedAt = c.UnitUpdatedAt.Time
	}
	return u
}

func flatRateColsFromGetRow(row sqlc.GetShippingTermRow) qtyJoinCols {
	return qtyJoinCols{
		QtyID:                 row.FlatRateQuantityID,
		Value:                 row.FlatRateValue,
		UnitID:                row.FlatRateUnitID,
		QtyCreatedAt:          row.FlatRateQuantityCreatedAt,
		QtyUpdatedAt:          row.FlatRateQuantityUpdatedAt,
		UnitName:              row.FlatRateUnitName,
		UnitAbbreviation:      row.FlatRateUnitAbbreviation,
		UnitType:              row.FlatRateUnitType,
		UnitRatioNumerator:    row.FlatRateUnitRatioNumerator,
		UnitRatioDenominator:  row.FlatRateUnitRatioDenominator,
		UnitOffsetNumerator:   row.FlatRateUnitOffsetNumerator,
		UnitOffsetDenominator: row.FlatRateUnitOffsetDenominator,
		UnitIsBaseUnit:        row.FlatRateUnitIsBaseUnit,
		UnitAccountID:         row.FlatRateUnitAccountID,
		UnitCreatedAt:         row.FlatRateUnitCreatedAt,
		UnitUpdatedAt:         row.FlatRateUnitUpdatedAt,
	}
}

func minimumOrderColsFromGetRow(row sqlc.GetShippingTermRow) qtyJoinCols {
	return qtyJoinCols{
		QtyID:                 row.MinimumOrderQuantityID,
		Value:                 row.MinimumOrderValue,
		UnitID:                row.MinimumOrderUnitID,
		QtyCreatedAt:          row.MinimumOrderQuantityCreatedAt,
		QtyUpdatedAt:          row.MinimumOrderQuantityUpdatedAt,
		UnitName:              row.MinimumOrderUnitName,
		UnitAbbreviation:      row.MinimumOrderUnitAbbreviation,
		UnitType:              row.MinimumOrderUnitType,
		UnitRatioNumerator:    row.MinimumOrderUnitRatioNumerator,
		UnitRatioDenominator:  row.MinimumOrderUnitRatioDenominator,
		UnitOffsetNumerator:   row.MinimumOrderUnitOffsetNumerator,
		UnitOffsetDenominator: row.MinimumOrderUnitOffsetDenominator,
		UnitIsBaseUnit:        row.MinimumOrderUnitIsBaseUnit,
		UnitAccountID:         row.MinimumOrderUnitAccountID,
		UnitCreatedAt:         row.MinimumOrderUnitCreatedAt,
		UnitUpdatedAt:         row.MinimumOrderUnitUpdatedAt,
	}
}

func carrierOptionToDomainServiceLevel(co sqlc.ListFreeShippingCarrierOptionsByShippingTermIDRow) *domain.ServiceLevel {
	return mapServiceLevelRow(sqlc.GetCarrierOptionRow(co))
}

func mapGetShippingTermRow(row sqlc.GetShippingTermRow) *domain.ShippingTerm {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.ShippingTerm{
		ID:                 row.ID,
		Name:               row.Name,
		Type:               shippingTermTypeFromBooleans(row.IsFreightExempt, row.IsCarrierRate),
		Status:             constants.ArchiveStatusFromIsActive(row.IsActive),
		FlatRate:           mapShippingTermQuantityFromCols(flatRateColsFromGetRow(row)),
		MinimumOrderValue:  mapShippingTermQuantityFromCols(minimumOrderColsFromGetRow(row)),
		FlatRateAmount:     storedMoneyPtr(row.FlatRateValue, row.FlatRateCurrency),
		MinimumOrderAmount: storedMoneyPtr(row.MinimumOrderValue, row.MinimumOrderValueCurrency),
		AccountID:          accountID,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func flatRateColsFromByIDsRow(row sqlc.GetShippingTermsByIDsRow) qtyJoinCols {
	return qtyJoinCols{
		QtyID:                 row.FlatRateQuantityID,
		Value:                 row.FlatRateValue,
		UnitID:                row.FlatRateUnitID,
		QtyCreatedAt:          row.FlatRateQuantityCreatedAt,
		QtyUpdatedAt:          row.FlatRateQuantityUpdatedAt,
		UnitName:              row.FlatRateUnitName,
		UnitAbbreviation:      row.FlatRateUnitAbbreviation,
		UnitType:              row.FlatRateUnitType,
		UnitRatioNumerator:    row.FlatRateUnitRatioNumerator,
		UnitRatioDenominator:  row.FlatRateUnitRatioDenominator,
		UnitOffsetNumerator:   row.FlatRateUnitOffsetNumerator,
		UnitOffsetDenominator: row.FlatRateUnitOffsetDenominator,
		UnitIsBaseUnit:        row.FlatRateUnitIsBaseUnit,
		UnitAccountID:         row.FlatRateUnitAccountID,
		UnitCreatedAt:         row.FlatRateUnitCreatedAt,
		UnitUpdatedAt:         row.FlatRateUnitUpdatedAt,
	}
}

func minimumOrderColsFromByIDsRow(row sqlc.GetShippingTermsByIDsRow) qtyJoinCols {
	return qtyJoinCols{
		QtyID:                 row.MinimumOrderQuantityID,
		Value:                 row.MinimumOrderValue,
		UnitID:                row.MinimumOrderUnitID,
		QtyCreatedAt:          row.MinimumOrderQuantityCreatedAt,
		QtyUpdatedAt:          row.MinimumOrderQuantityUpdatedAt,
		UnitName:              row.MinimumOrderUnitName,
		UnitAbbreviation:      row.MinimumOrderUnitAbbreviation,
		UnitType:              row.MinimumOrderUnitType,
		UnitRatioNumerator:    row.MinimumOrderUnitRatioNumerator,
		UnitRatioDenominator:  row.MinimumOrderUnitRatioDenominator,
		UnitOffsetNumerator:   row.MinimumOrderUnitOffsetNumerator,
		UnitOffsetDenominator: row.MinimumOrderUnitOffsetDenominator,
		UnitIsBaseUnit:        row.MinimumOrderUnitIsBaseUnit,
		UnitAccountID:         row.MinimumOrderUnitAccountID,
		UnitCreatedAt:         row.MinimumOrderUnitCreatedAt,
		UnitUpdatedAt:         row.MinimumOrderUnitUpdatedAt,
	}
}

func mapByIDsShippingTermRow(row sqlc.GetShippingTermsByIDsRow) *domain.ShippingTerm {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.ShippingTerm{
		ID:                 row.ID,
		Name:               row.Name,
		Type:               shippingTermTypeFromBooleans(row.IsFreightExempt, row.IsCarrierRate),
		Status:             constants.ArchiveStatusFromIsActive(row.IsActive),
		FlatRate:           mapShippingTermQuantityFromCols(flatRateColsFromByIDsRow(row)),
		MinimumOrderValue:  mapShippingTermQuantityFromCols(minimumOrderColsFromByIDsRow(row)),
		FlatRateAmount:     storedMoneyPtr(row.FlatRateValue, row.FlatRateCurrency),
		MinimumOrderAmount: storedMoneyPtr(row.MinimumOrderValue, row.MinimumOrderValueCurrency),
		AccountID:          accountID,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func (r *shippingTermRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetShippingTermsByIDs(ctx, sqlc.GetShippingTermsByIDsParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.ShippingTerm, len(rows))
	for i, row := range rows {
		st := mapByIDsShippingTermRow(row)
		if apiErr := r.loadFreeShippingRules(ctx, st); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		out[i] = st
	}
	return out, nil
}

func (r *shippingTermRepoImpl) loadFreeShippingRules(ctx context.Context, st *domain.ShippingTerm) *apierror.APIError {
	opts, err := r.queries.ListFreeShippingCarrierOptionsByShippingTermID(ctx, st.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	st.FreeShippingServiceLevels = make([]*domain.ServiceLevel, len(opts))
	st.FreeShippingServiceLevelIDs = make([]string, len(opts))
	for i := range opts {
		sl := carrierOptionToDomainServiceLevel(opts[i])
		st.FreeShippingServiceLevels[i] = sl
		st.FreeShippingServiceLevelIDs[i] = sl.ID
	}
	return nil
}

// shippingTermListSelect matches GetShippingTerm's columns, so a page scans into GetShippingTermRow.
const shippingTermListSelect = `SELECT st.id, st.name, st.is_freight_exempt, st.is_carrier_rate, st.is_active, st.account_id,` +
	` st.flat_rate_id, st.minimum_order_id, st.flat_rate_currency, st.minimum_order_value_currency, st.created_at, st.updated_at,` +
	` fr.id, fr.value, fr.unit_id, fr.created_at, fr.updated_at,` +
	` fr_u.name, fr_u.abbreviation, fr_u.unit_dimension_code, fr_u.ratio_numerator, fr_u.ratio_denominator,` +
	` fr_u.offset_numerator, fr_u.offset_denominator, fr_u.is_base_unit, fr_u.account_id, fr_u.created_at, fr_u.updated_at,` +
	` mo.id, mo.value, mo.unit_id, mo.created_at, mo.updated_at,` +
	` mo_u.name, mo_u.abbreviation, mo_u.unit_dimension_code, mo_u.ratio_numerator, mo_u.ratio_denominator,` +
	` mo_u.offset_numerator, mo_u.offset_denominator, mo_u.is_base_unit, mo_u.account_id, mo_u.created_at, mo_u.updated_at` +
	` FROM shipping_term st` +
	` LEFT JOIN quantity fr ON st.flat_rate_id = fr.id` +
	` LEFT JOIN unit fr_u ON fr.unit_id = fr_u.id` +
	` LEFT JOIN quantity mo ON st.minimum_order_id = mo.id` +
	` LEFT JOIN unit mo_u ON mo.unit_id = mo_u.id`

func scanShippingTermListRow(rows *gosql.Rows) (*domain.ShippingTerm, error) {
	var i sqlc.GetShippingTermRow
	err := rows.Scan(
		&i.ID, &i.Name, &i.IsFreightExempt, &i.IsCarrierRate, &i.IsActive, &i.AccountID,
		&i.FlatRateID, &i.MinimumOrderID, &i.FlatRateCurrency, &i.MinimumOrderValueCurrency, &i.CreatedAt, &i.UpdatedAt,
		&i.FlatRateQuantityID, &i.FlatRateValue, &i.FlatRateUnitID, &i.FlatRateQuantityCreatedAt, &i.FlatRateQuantityUpdatedAt,
		&i.FlatRateUnitName, &i.FlatRateUnitAbbreviation, &i.FlatRateUnitType, &i.FlatRateUnitRatioNumerator, &i.FlatRateUnitRatioDenominator,
		&i.FlatRateUnitOffsetNumerator, &i.FlatRateUnitOffsetDenominator, &i.FlatRateUnitIsBaseUnit, &i.FlatRateUnitAccountID,
		&i.FlatRateUnitCreatedAt, &i.FlatRateUnitUpdatedAt,
		&i.MinimumOrderQuantityID, &i.MinimumOrderValue, &i.MinimumOrderUnitID, &i.MinimumOrderQuantityCreatedAt, &i.MinimumOrderQuantityUpdatedAt,
		&i.MinimumOrderUnitName, &i.MinimumOrderUnitAbbreviation, &i.MinimumOrderUnitType, &i.MinimumOrderUnitRatioNumerator, &i.MinimumOrderUnitRatioDenominator,
		&i.MinimumOrderUnitOffsetNumerator, &i.MinimumOrderUnitOffsetDenominator, &i.MinimumOrderUnitIsBaseUnit, &i.MinimumOrderUnitAccountID,
		&i.MinimumOrderUnitCreatedAt, &i.MinimumOrderUnitUpdatedAt,
	)
	return mapGetShippingTermRow(i), err
}

func (r *shippingTermRepoImpl) List(ctx context.Context, params domain.ListShippingTermsParams) (*domain.ListShippingTermsResult, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.list")
	defer span.End()

	cur, apiErr := decodeLookupCursor(params.Cursor)
	if apiErr != nil {
		return nil, apiErr
	}
	q := lookupListQuery{selectFrom: shippingTermListSelect, alias: "st"}
	q.and("(st.account_id = ? OR st.account_id IS NULL)", params.AccountID)
	if search := buildShippingTermSearchParams(params.Query); search.Valid {
		q.and("st.name LIKE ?", search.String)
	}
	if params.IsActive != nil {
		q.and("st.is_active = ?", *params.IsActive)
	}
	query, args := q.sql(cur, params.Limit+1)
	terms, err := queryLookupRows(ctx, r.queries.DB(), query, args, scanShippingTermListRow)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if slices.Contains(params.Includes, "free_shipping_service_levels") {
		for _, t := range terms {
			if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}
	result, pageInfo := pagination.BuildPageString(terms, params.Limit, cursorDirection(cur), shippingTermCreatedAt, shippingTermID)
	return &domain.ListShippingTermsResult{ShippingTerms: result, PageInfo: pageInfo}, nil
}

func (r *shippingTermRepoImpl) Get(ctx context.Context, params domain.GetShippingTermParams) (*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.get")
	defer span.End()

	row, err := r.queries.GetShippingTerm(ctx, sqlc.GetShippingTermParams{
		ID:        params.ShippingTermID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	st := mapGetShippingTermRow(row)
	if slices.Contains(params.Includes, "free_shipping_service_levels") {
		if apiErr := r.loadFreeShippingRules(ctx, st); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return st, nil
}

func (r *shippingTermRepoImpl) Create(ctx context.Context, id string, params domain.CreateShippingTermParams) (*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.create")
	defer span.End()

	isFreightExempt, isCarrierRate := shippingTermTypeToBooleans(params.Type)
	insertParams := sqlc.InsertShippingTermParams{
		ID:              id,
		Name:            params.Name,
		IsFreightExempt: isFreightExempt,
		IsCarrierRate:   isCarrierRate,
		IsActive:        params.Status != constants.ArchiveStatusArchived,
		AccountID:       gosql.NullString{String: params.AccountID, Valid: true},
	}
	if params.FlatRateID != nil {
		insertParams.FlatRateID = gosql.NullString{String: *params.FlatRateID, Valid: true}
	}
	if params.MinimumOrderID != nil {
		insertParams.MinimumOrderID = gosql.NullString{String: *params.MinimumOrderID, Valid: true}
	}
	err := r.queries.InsertShippingTerm(ctx, insertParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := db.MapSQLError(syncShippingTermMoney(ctx, r.queries.DB(), id)); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetShippingTermParams{AccountID: params.AccountID, ShippingTermID: id, Includes: params.Includes})
}

func (r *shippingTermRepoImpl) Update(ctx context.Context, params domain.UpdateShippingTermParams) (*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.update")
	defer span.End()

	var isFreightExempt, isCarrierRate gosql.NullBool
	if params.Type != nil {
		fe, cr := shippingTermTypeToBooleans(*params.Type)
		isFreightExempt = gosql.NullBool{Bool: fe, Valid: true}
		isCarrierRate = gosql.NullBool{Bool: cr, Valid: true}
	}
	updateParams := sqlc.UpdateShippingTermParams{
		ID:              params.ShippingTermID,
		AccountID:       gosql.NullString{String: params.AccountID, Valid: true},
		Name:            toNullString(params.Name),
		IsFreightExempt: isFreightExempt,
		IsCarrierRate:   isCarrierRate,
		IsActive:        archiveStatusToNullBool(params.Status),
	}
	if params.FlatRateID != nil {
		updateParams.FlatRateID = gosql.NullString{String: *params.FlatRateID, Valid: true}
	}
	if params.MinimumOrderID != nil {
		updateParams.MinimumOrderID = gosql.NullString{String: *params.MinimumOrderID, Valid: true}
	}
	result, err := r.queries.UpdateShippingTerm(ctx, updateParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Shipping term not found."))
	}
	if apiErr := db.MapSQLError(syncShippingTermMoney(ctx, r.queries.DB(), params.ShippingTermID)); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetShippingTermParams{AccountID: params.AccountID, ShippingTermID: params.ShippingTermID, Includes: params.Includes})
}

func (r *shippingTermRepoImpl) Delete(ctx context.Context, params domain.DeleteShippingTermParams) *apierror.APIError {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.delete")
	defer span.End()

	result, err := r.queries.DeleteShippingTerm(ctx, sqlc.DeleteShippingTermParams{
		ID:        params.ShippingTermID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewNotFoundError("Shipping term not found."))
	}

	return nil
}

func (r *shippingTermRepoImpl) InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.insert_quantity")
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

func (r *shippingTermRepoImpl) UpdateQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.update_quantity")
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

func (r *shippingTermRepoImpl) DeleteQuantity(ctx context.Context, id string) *apierror.APIError {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.delete_quantity")
	defer span.End()

	err := r.queries.DeleteQuantity(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingTermRepoImpl) InsertFreeShippingRule(ctx context.Context, id, shippingTermID, serviceLevelID string) *apierror.APIError {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.insert_free_shipping_rule")
	defer span.End()

	err := r.queries.InsertFreeShippingRule(ctx, sqlc.InsertFreeShippingRuleParams{
		ID:              id,
		ShippingTermID:  shippingTermID,
		CarrierOptionID: serviceLevelID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingTermRepoImpl) DeleteFreeShippingRulesByShippingTermID(ctx context.Context, shippingTermID string) *apierror.APIError {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.delete_free_shipping_rules")
	defer span.End()

	err := r.queries.DeleteFreeShippingRulesByShippingTermID(ctx, shippingTermID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingTermRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.exists_by_name")
	defer span.End()

	var count int64
	var err error
	if exclude := toNullString(excludeID); exclude.Valid {
		count, err = r.queries.CountOtherShippingTermsByName(ctx, sqlc.CountOtherShippingTermsByNameParams{
			Name:      name,
			AccountID: gosql.NullString{String: accountID, Valid: true},
			ExcludeID: exclude.String,
		})
	} else {
		count, err = r.queries.CountShippingTermsByName(ctx, sqlc.CountShippingTermsByNameParams{
			Name:      name,
			AccountID: gosql.NullString{String: accountID, Valid: true},
		})
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
