package repository

import (
	"context"
	gosql "database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	idgen "github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var productLineRepoTracer = tracing.GetTracer("core-service.product_line_repository")

type productLineRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductLineRepo(queries *sqlc.Queries) domain.ProductLineRepo {
	return &productLineRepoImpl{queries: queries}
}

func productLineCreatedAt(pl *domain.ProductLineFull) time.Time { return pl.CreatedAt }
func productLineID(pl *domain.ProductLineFull) string           { return pl.ID }

func buildProductLineSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

func mapProductLineForwardRow(row sqlc.ListProductLinesForwardRow) *domain.ProductLineFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	lotID, lotValue, lotUnitID := lotFieldsOf(row.DefaultLotID, row.DefaultLotValue, row.DefaultLotUnitID)

	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
		DefaultLotID:     lotID,
		DefaultLotValue:  lotValue,
		DefaultLotUnitID: lotUnitID,
		AccountID:        accountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

// lotFieldsOf reads a product line's lot off any of the row shapes that project it. The three move together: a value with no unit cannot say whether 60 means pairs or eaches, so a half-joined row reads as no convention rather than being guessed at.
func lotFieldsOf(id, value, unitID gosql.NullString) (*string, *string, *string) {
	if !id.Valid || !value.Valid || !unitID.Valid || unitID.String == "" {
		return nil, nil, nil
	}
	lotID, lotValue, lotUnit := id.String, value.String, unitID.String
	return &lotID, &lotValue, &lotUnit
}

func mapProductLineBackwardRow(row sqlc.ListProductLinesBackwardRow) *domain.ProductLineFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	lotID, lotValue, lotUnitID := lotFieldsOf(row.DefaultLotID, row.DefaultLotValue, row.DefaultLotUnitID)

	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
		DefaultLotID:     lotID,
		DefaultLotValue:  lotValue,
		DefaultLotUnitID: lotUnitID,
		AccountID:        accountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapGetProductLineRow(row sqlc.GetProductLineRow) *domain.ProductLineFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	lotID, lotValue, lotUnitID := lotFieldsOf(row.DefaultLotID, row.DefaultLotValue, row.DefaultLotUnitID)

	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
		DefaultLotID:     lotID,
		DefaultLotValue:  lotValue,
		DefaultLotUnitID: lotUnitID,
		AccountID:        accountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func commissionPolicyToNullBool(p *constants.CommissionPolicy) gosql.NullBool {
	if p == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: p.ToBool(), Valid: true}
}

func freightPolicyToNullBool(p *constants.FreightPolicy) gosql.NullBool {
	if p == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: p.ToBool(), Valid: true}
}

func (r *productLineRepoImpl) List(ctx context.Context, params domain.ListProductLinesParams) (*domain.ListProductLinesResult, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.list")
	defer span.End()

	searchQuery := buildProductLineSearchParams(params.Query)
	accountID := gosql.NullString{String: params.AccountID, Valid: true}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListProductLinesBackward(ctx, sqlc.ListProductLinesBackwardParams{
				AccountID:       accountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			items := make([]*domain.ProductLineFull, len(rows))
			for i, row := range rows {
				items[i] = mapProductLineBackwardRow(row)
			}
			result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, productLineCreatedAt, productLineID)
			return &domain.ListProductLinesResult{ProductLines: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListProductLinesForward(ctx, sqlc.ListProductLinesForwardParams{
			AccountID:       accountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		items := make([]*domain.ProductLineFull, len(rows))
		for i, row := range rows {
			items[i] = mapProductLineForwardRow(row)
		}
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, productLineCreatedAt, productLineID)
		return &domain.ListProductLinesResult{ProductLines: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListProductLinesForward(ctx, sqlc.ListProductLinesForwardParams{
		AccountID:   accountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ProductLineFull, len(rows))
	for i, row := range rows {
		items[i] = mapProductLineForwardRow(row)
	}
	result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, productLineCreatedAt, productLineID)
	return &domain.ListProductLinesResult{ProductLines: result, PageInfo: pageInfo}, nil
}

func (r *productLineRepoImpl) Get(ctx context.Context, params domain.GetProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get")
	defer span.End()

	row, err := r.queries.GetProductLine(ctx, sqlc.GetProductLineParams{
		ID:        params.ProductLineID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetProductLineRow(row), nil
}

func (r *productLineRepoImpl) Create(ctx context.Context, id string, params domain.CreateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.create")
	defer span.End()

	// The lot is its own quantity row, written before the line that points at it.
	lotID := gosql.NullString{}
	if params.DefaultLot != nil {
		newID, apiErr := idgen.GenID(idgen.QuantityIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if err := r.queries.InsertProductLineDefaultLotQuantity(ctx, sqlc.InsertProductLineDefaultLotQuantityParams{
			ID:     newID,
			Value:  params.DefaultLot.Value,
			UnitID: params.DefaultLot.UnitID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		lotID = gosql.NullString{String: newID, Valid: true}
	}

	err := r.queries.InsertProductLine(ctx, sqlc.InsertProductLineParams{
		ID:                 id,
		Name:               params.Name,
		IsCommissionExempt: params.CommissionPolicy.ToBool(),
		IsFreightExempt:    params.FreightPolicy.ToBool(),
		UnitGroupID:        params.UnitGroupID,
		DefaultLotID:       lotID,
		AccountID:          gosql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetProductLineParams{AccountID: params.AccountID, ProductLineID: id})
}

func (r *productLineRepoImpl) Update(ctx context.Context, params domain.UpdateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.update")
	defer span.End()

	// The line's existing lot row is updated in place rather than replaced, so nothing pointing at it is orphaned and the id stays stable across edits.
	existingLotID, err := r.queries.GetProductLineDefaultLotID(ctx, sqlc.GetProductLineDefaultLotIDParams{
		ID:        params.ProductLineID,
		AccountID: gosql.NullString{String: params.AccountID, Valid: true},
	})
	if err != nil && err != gosql.ErrNoRows {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	lotID := gosql.NullString{}
	switch {
	case params.ClearDefaultLot:
		if existingLotID.Valid {
			if err := r.queries.DeleteProductLineDefaultLotQuantity(ctx, existingLotID.String); err != nil {
				if apiErr := db.MapSQLError(err); apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
			}
		}
	case params.DefaultLot != nil && existingLotID.Valid:
		if err := r.queries.UpdateProductLineDefaultLotQuantity(ctx, sqlc.UpdateProductLineDefaultLotQuantityParams{
			ID:     existingLotID.String,
			Value:  params.DefaultLot.Value,
			UnitID: params.DefaultLot.UnitID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		lotID = existingLotID
	case params.DefaultLot != nil:
		newID, apiErr := idgen.GenID(idgen.QuantityIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if err := r.queries.InsertProductLineDefaultLotQuantity(ctx, sqlc.InsertProductLineDefaultLotQuantityParams{
			ID:     newID,
			Value:  params.DefaultLot.Value,
			UnitID: params.DefaultLot.UnitID,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		lotID = gosql.NullString{String: newID, Valid: true}
	}

	result, err := r.queries.UpdateProductLine(ctx, sqlc.UpdateProductLineParams{
		ID:                 params.ProductLineID,
		AccountID:          gosql.NullString{String: params.AccountID, Valid: true},
		Name:               toNullString(params.Name),
		IsCommissionExempt: commissionPolicyToNullBool(params.CommissionPolicy),
		IsFreightExempt:    freightPolicyToNullBool(params.FreightPolicy),
		UnitGroupID:        toNullString(params.UnitGroupID),
		DefaultLotID:       lotID,
		ClearDefaultLot:    params.ClearDefaultLot,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product line not found."))
	}

	return r.Get(ctx, domain.GetProductLineParams{AccountID: params.AccountID, ProductLineID: params.ProductLineID})
}

func (r *productLineRepoImpl) Delete(ctx context.Context, params domain.DeleteProductLineParams) *apierror.APIError {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.delete")
	defer span.End()

	result, err := r.queries.DeleteProductLine(ctx, sqlc.DeleteProductLineParams{
		ID:        params.ProductLineID,
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
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Product line not found."))
	}

	return nil
}

func (r *productLineRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.exists_by_name")
	defer span.End()

	count, err := r.queries.CountProductLinesByName(ctx, sqlc.CountProductLinesByNameParams{
		Name:      name,
		AccountID: gosql.NullString{String: accountID, Valid: true},
		ExcludeID: toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *productLineRepoImpl) GetUnitGroup(ctx context.Context, accountID, unitGroupID string, includes []string) (*domain.ProductLineUnitGroup, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_unit_group")
	defer span.End()

	row, err := r.queries.GetUnitGroupForProductLine(ctx, sqlc.GetUnitGroupForProductLineParams{
		ID:        unitGroupID,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ug := &domain.ProductLineUnitGroup{
		ID:         row.ID,
		Name:       row.Name,
		BaseUnitID: row.BaseUnitID,
		Type:       row.UnitTypeCode,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}

	// Includes may nest further (e.g. product_line.unit_group.associated_units.unit).
	incCoversFragment := func(fragments ...string) bool {
		for _, inc := range includes {
			for _, frag := range fragments {
				if inc == frag || strings.HasPrefix(inc, frag+".") {
					return true
				}
			}
		}
		return false
	}

	wantsBaseUnit := incCoversFragment("product_line.unit_group.base_unit", "unit_group.base_unit")
	if wantsBaseUnit && ug.BaseUnitID != "" {
		unitRows, err := r.queries.GetUnitsByIDs(ctx, []string{ug.BaseUnitID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(unitRows) > 0 {
			lu := mapGetUnitsByIDsRowToLightUnit(unitRows[0])
			ug.BaseUnit = &lu
		}
	}

	wantsAssociatedUnits := incCoversFragment("product_line.unit_group.associated_units", "unit_group.associated_units")
	if wantsAssociatedUnits {
		ugUnitRows, err := r.queries.ListUnitGroupUnitsByUnitGroupIDs(ctx, []string{ug.ID})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		units := make([]*domain.UnitGroupUnit, len(ugUnitRows))
		for i, ugRow := range ugUnitRows {
			units[i] = mapUnitGroupUnitsByUnitGroupIDsRow(ugRow)
		}
		ug.AssociatedUnits = units
	}

	return ug, nil
}

func (r *productLineRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetProductLinesByIDsScoped(ctx, sqlc.GetProductLinesByIDsScopedParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.ProductLineFull, len(rows))
	for i, row := range rows {
		out[i] = mapGetProductLinesByIDsScopedRow(row)
	}
	// Stitch unit group data — always include base_unit and associated_units so the API gateway's SubField resolver has everything it needs.
	for _, pl := range out {
		ug, apiErr := r.GetUnitGroup(ctx, accountID, pl.UnitGroupID, []string{"unit_group.base_unit", "unit_group.associated_units"})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		pl.UnitGroup = ug
	}
	return out, nil
}

func mapGetProductLinesByIDsScopedRow(row sqlc.GetProductLinesByIDsScopedRow) *domain.ProductLineFull {
	var accountID *string
	if row.AccountID.Valid {
		accountID = &row.AccountID.String
	}
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}
	lotID, lotValue, lotUnitID := lotFieldsOf(row.DefaultLotID, row.DefaultLotValue, row.DefaultLotUnitID)

	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
		DefaultLotID:     lotID,
		DefaultLotValue:  lotValue,
		DefaultLotUnitID: lotUnitID,
		AccountID:        accountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (r *productLineRepoImpl) IsUnitInGroup(ctx context.Context, unitGroupID, unitID string) (bool, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.is_unit_in_group")
	defer span.End()

	count, err := r.queries.CountUnitInGroup(ctx, sqlc.CountUnitInGroupParams{
		UnitID:      unitID,
		UnitGroupID: unitGroupID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *productLineRepoImpl) GetItemLotOverride(ctx context.Context, accountID, itemID string) (float64, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_item_lot_override")
	defer span.End()

	value, err := r.queries.GetItemLotOverride(ctx, sqlc.GetItemLotOverrideParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if err == gosql.ErrNoRows {
		// No per-item setting is the normal case, not an error.
		return 0, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return decimalToFloat64(value), nil
}

func (r *productLineRepoImpl) GetProductLineLotForItem(ctx context.Context, accountID, itemID string) (*domain.ProductLineLotDefault, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_lot_for_item")
	defer span.End()

	row, err := r.queries.GetProductLineForItem(ctx, sqlc.GetProductLineForItemParams{
		ItemID:    itemID,
		AccountID: accountID,
	})
	if err == gosql.ErrNoRows {
		// An intermediate item has no product row, so no line of its own.
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The query inner-joins the quantity, so a row here always carries a full lot.
	quantity, err := strconv.ParseFloat(row.DefaultLotValue, 64)
	if err != nil || quantity <= 0 {
		return nil, nil
	}
	return &domain.ProductLineLotDefault{
		ProductLineID: row.ID,
		Quantity:      quantity,
		UnitID:        row.DefaultLotUnitID,
	}, nil
}

func (r *productLineRepoImpl) GetDownstreamProductLineLot(ctx context.Context, accountID, itemID string) (*domain.ProductLineLotDefault, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_downstream_lot")
	defer span.End()

	rows, err := r.queries.ResolveItemLotFromDownstream(ctx, sqlc.ResolveItemLotFromDownstreamParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Ordered highest-demand first with the line id breaking ties, so the same item always resolves the same way.
	for _, row := range rows {
		quantity, err := strconv.ParseFloat(row.DefaultLotValue, 64)
		if err != nil || quantity <= 0 {
			continue
		}
		return &domain.ProductLineLotDefault{
			ProductLineID: row.ProductLineID,
			Quantity:      quantity,
			UnitID:        row.DefaultLotUnitID,
		}, nil
	}
	return nil, nil
}
