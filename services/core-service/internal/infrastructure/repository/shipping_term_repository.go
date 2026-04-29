package repository

import (
	"context"
	gosql "database/sql"
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

func mapShippingTermQuantity(id, value, unitID, unitAbbreviation, unitType gosql.NullString) *domain.Quantity {
	if !id.Valid {
		return nil
	}
	return &domain.Quantity{
		ID:               id.String,
		Value:            value.String,
		UnitID:           unitID.String,
		UnitAbbreviation: unitAbbreviation.String,
		UnitType:         unitType.String,
	}
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
		FlatRate:          mapShippingTermQuantity(row.FlatRateQuantityID, row.FlatRateValue, row.FlatRateUnitID, row.FlatRateUnitAbbreviation, row.FlatRateUnitType),
		MinimumOrderValue: mapShippingTermQuantity(row.MinimumOrderQuantityID, row.MinimumOrderValue, row.MinimumOrderUnitID, row.MinimumOrderUnitAbbreviation, row.MinimumOrderUnitType),
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
		FlatRate:          mapShippingTermQuantity(row.FlatRateQuantityID, row.FlatRateValue, row.FlatRateUnitID, row.FlatRateUnitAbbreviation, row.FlatRateUnitType),
		MinimumOrderValue: mapShippingTermQuantity(row.MinimumOrderQuantityID, row.MinimumOrderValue, row.MinimumOrderUnitID, row.MinimumOrderUnitAbbreviation, row.MinimumOrderUnitType),
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
		FlatRate:          mapShippingTermQuantity(row.FlatRateQuantityID, row.FlatRateValue, row.FlatRateUnitID, row.FlatRateUnitAbbreviation, row.FlatRateUnitType),
		MinimumOrderValue: mapShippingTermQuantity(row.MinimumOrderQuantityID, row.MinimumOrderValue, row.MinimumOrderUnitID, row.MinimumOrderUnitAbbreviation, row.MinimumOrderUnitType),
		AccountID:         accountID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func (r *shippingTermRepoImpl) loadFreeShippingRules(ctx context.Context, st *domain.ShippingTerm) *apierror.APIError {
	rules, err := r.queries.ListFreeShippingRulesByShippingTermID(ctx, st.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	ids := make([]string, len(rules))
	for i, rule := range rules {
		ids[i] = rule.CarrierOptionID
	}
	st.FreeShippingServiceLevelIDs = ids
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
			for _, t := range terms {
				if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
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
		for _, t := range terms {
			if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
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
	for _, t := range terms {
		if apiErr := r.loadFreeShippingRules(ctx, t); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
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
	if apiErr := r.loadFreeShippingRules(ctx, st); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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

	return r.Get(ctx, domain.GetShippingTermParams{AccountID: params.AccountID, ShippingTermID: id})
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

	return r.Get(ctx, domain.GetShippingTermParams{AccountID: params.AccountID, ShippingTermID: params.ShippingTermID})
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
