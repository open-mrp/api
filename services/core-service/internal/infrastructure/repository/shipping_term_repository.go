package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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
		return constants.ShippingTermTypeFreeFreight
	}
	if isCarrierRate {
		return constants.ShippingTermTypeCarrierRateFreight
	}
	return constants.ShippingTermTypeFlatRateFreight
}

// shippingTermTypeToBooleans maps a ShippingTermType to DB booleans (isFreightExempt, isCarrierRate).
func shippingTermTypeToBooleans(t constants.ShippingTermType) (bool, bool) {
	switch t {
	case constants.ShippingTermTypeFreeFreight:
		return true, false
	case constants.ShippingTermTypeCarrierRateFreight:
		return false, true
	default: // flat_rate_freight
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

func flatRateColsFromForwardRow(row sqlc.ListShippingTermsForwardRow) qtyJoinCols {
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

func minimumOrderColsFromForwardRow(row sqlc.ListShippingTermsForwardRow) qtyJoinCols {
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

func flatRateColsFromBackwardRow(row sqlc.ListShippingTermsBackwardRow) qtyJoinCols {
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

func minimumOrderColsFromBackwardRow(row sqlc.ListShippingTermsBackwardRow) qtyJoinCols {
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

func carrierOptionToDomainServiceLevel(co sqlc.CarrierOption) *domain.ServiceLevel {
	sl := &domain.ServiceLevel{
		ID:              co.ID,
		Name:            co.Name,
		Code:            co.Code,
		IsPortalEnabled: co.IsPortalEnabled,
		IsDefault:       co.IsDefault,
		CarrierID:       co.CarrierID,
		CreatedAt:       co.CreatedAt,
		UpdatedAt:       co.UpdatedAt,
	}
	if co.ServiceLevelToken.Valid {
		t := co.ServiceLevelToken.String
		sl.ServiceLevelToken = &t
	}
	if co.AccountID.Valid {
		a := co.AccountID.String
		sl.AccountID = &a
	}
	return sl
}

func mapForwardShippingTermRow(row sqlc.ListShippingTermsForwardRow) *domain.ShippingTerm {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.ShippingTerm{
		ID:                row.ID,
		Name:              row.Name,
		Type:              shippingTermTypeFromBooleans(row.IsFreightExempt, row.IsCarrierRate),
		FlatRate:          mapShippingTermQuantityFromCols(flatRateColsFromForwardRow(row)),
		MinimumOrderValue: mapShippingTermQuantityFromCols(minimumOrderColsFromForwardRow(row)),
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapBackwardShippingTermRow(row sqlc.ListShippingTermsBackwardRow) *domain.ShippingTerm {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.ShippingTerm{
		ID:                row.ID,
		Name:              row.Name,
		Type:              shippingTermTypeFromBooleans(row.IsFreightExempt, row.IsCarrierRate),
		FlatRate:          mapShippingTermQuantityFromCols(flatRateColsFromBackwardRow(row)),
		MinimumOrderValue: mapShippingTermQuantityFromCols(minimumOrderColsFromBackwardRow(row)),
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapGetShippingTermRow(row sqlc.GetShippingTermRow) *domain.ShippingTerm {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	return &domain.ShippingTerm{
		ID:                row.ID,
		Name:              row.Name,
		Type:              shippingTermTypeFromBooleans(row.IsFreightExempt, row.IsCarrierRate),
		FlatRate:          mapShippingTermQuantityFromCols(flatRateColsFromGetRow(row)),
		MinimumOrderValue: mapShippingTermQuantityFromCols(minimumOrderColsFromGetRow(row)),
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
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
		ID:                row.ID,
		Name:              row.Name,
		Type:              shippingTermTypeFromBooleans(row.IsFreightExempt, row.IsCarrierRate),
		FlatRate:          mapShippingTermQuantityFromCols(flatRateColsFromByIDsRow(row)),
		MinimumOrderValue: mapShippingTermQuantityFromCols(minimumOrderColsFromByIDsRow(row)),
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
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

func (r *shippingTermRepoImpl) List(ctx context.Context, params domain.ListShippingTermsParams) (*domain.ListShippingTermsResult, *apierror.APIError) {
	ctx, span := shippingTermRepoTracer.Start(ctx, "repository.shipping_term.list")
	defer span.End()

	searchQuery := buildShippingTermSearchParams(params.Query)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListShippingTermsBackward(ctx, sqlc.ListShippingTermsBackwardParams{
				AccountID:       accountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			terms := make([]*domain.ShippingTerm, len(rows))
			for i, row := range rows {
				terms[i] = mapBackwardShippingTermRow(row)
			}
			if slices.Contains(params.Includes, "free_shipping_service_levels") {
				for _, t := range terms {
					if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}
				}
			}
			result, pageInfo := pagination.BuildPageString(terms, params.Limit, cursorDir, shippingTermCreatedAt, shippingTermID)
			return &domain.ListShippingTermsResult{ShippingTerms: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListShippingTermsForward(ctx, sqlc.ListShippingTermsForwardParams{
			AccountID:       accountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		terms := make([]*domain.ShippingTerm, len(rows))
		for i, row := range rows {
			terms[i] = mapForwardShippingTermRow(row)
		}
		if slices.Contains(params.Includes, "free_shipping_service_levels") {
			for _, t := range terms {
				if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}
		result, pageInfo := pagination.BuildPageString(terms, params.Limit, cursorDir, shippingTermCreatedAt, shippingTermID)
		return &domain.ListShippingTermsResult{ShippingTerms: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListShippingTermsForward(ctx, sqlc.ListShippingTermsForwardParams{
		AccountID:   accountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	terms := make([]*domain.ShippingTerm, len(rows))
	for i, row := range rows {
		terms[i] = mapForwardShippingTermRow(row)
	}
	if slices.Contains(params.Includes, "free_shipping_service_levels") {
		for _, t := range terms {
			if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
	}
	result, pageInfo := pagination.BuildPageString(terms, params.Limit, cursorDir, shippingTermCreatedAt, shippingTermID)
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
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipping term not found."))
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Shipping term not found."))
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
