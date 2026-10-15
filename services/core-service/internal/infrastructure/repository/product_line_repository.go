package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
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

// mapProductLineRow maps any of the product line row shapes, which all project the same columns.
func mapProductLineRow(row sqlc.GetProductLineRow) *domain.ProductLineFull {
	pl := &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      nullStringToPtr(row.Description),
		Notes:            nullStringToPtr(row.Notes),
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID.String,
		Status:           constants.ArchiveStatusFromIsActive(row.IsActive),
		AccountID:        nullStringToPtr(row.AccountID),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if row.SupplyCode.Valid && row.SupplyCode.String != "" {
		supply := constants.Supply(row.SupplyCode.String)
		pl.Supply = &supply
	}
	return pl
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

func (r *productLineRepoImpl) Export(ctx context.Context, params domain.ExportProductLinesParams) ([]*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.export")
	defer span.End()

	rows, err := r.queries.ExportProductLines(ctx, sqlc.ExportProductLinesParams{
		AccountID:   gosql.NullString{String: params.AccountID, Valid: true},
		SearchQuery: buildProductLineSearchParams(params.Query),
		Limit:       exportQueryLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.ProductLineFull, len(rows))
	for i, row := range rows {
		line := &domain.ProductLineFull{
			ID:               row.ID,
			Name:             row.Name,
			CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
			FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		}
		if row.UnitGroupName.Valid {
			line.UnitGroup = &domain.ProductLineUnitGroup{Name: row.UnitGroupName.String}
		}
		lines[i] = line
	}
	return lines, nil
}

func (r *productLineRepoImpl) List(ctx context.Context, params domain.ListProductLinesParams) (*domain.ListProductLinesResult, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.list")
	defer span.End()

	filter := productLineListFilter{
		accountID: params.AccountID,
		search:    buildProductLineSearchParams(params.Query).String,
		isActive:  params.IsActive,
		dir:       pagination.DirectionForward,
		limit:     params.Limit + 1,
	}
	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction
		filter.dir = cur.Direction
		filter.cursorCreatedAt = &cur.OccurredAt
		filter.cursorID = cur.ID
	}

	query, args := buildProductLineListQuery(filter)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer rows.Close()
	scanned, err := scanProductLineListRows(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.ProductLineFull, len(scanned))
	for i, row := range scanned {
		items[i] = mapProductLineRow(row)
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

	return mapProductLineRow(row), nil
}

func (r *productLineRepoImpl) Create(ctx context.Context, id string, params domain.CreateProductLineParams) (*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.create")
	defer span.End()

	status := params.Status
	if status == "" {
		status = constants.ArchiveStatusActive
	}
	err := r.queries.InsertProductLine(ctx, sqlc.InsertProductLineParams{
		ID:                 id,
		Name:               params.Name,
		Description:        toNullString(params.Description),
		IsCommissionExempt: params.CommissionPolicy.ToBool(),
		IsFreightExempt:    params.FreightPolicy.ToBool(),
		UnitGroupID:        toNullString(params.UnitGroupID),
		SupplyCode:         toNullString(params.Supply.StringPtr()),
		IsActive:           status.IsActive(),
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

	isActive := gosql.NullBool{}
	if params.Status != nil {
		isActive = gosql.NullBool{Bool: params.Status.IsActive(), Valid: true}
	}
	supply := gosql.NullString{}
	if v, ok := params.Supply.Value(); ok {
		supply = gosql.NullString{String: string(v), Valid: true}
	}

	result, err := r.queries.UpdateProductLine(ctx, sqlc.UpdateProductLineParams{
		ID:                 params.ProductLineID,
		AccountID:          gosql.NullString{String: params.AccountID, Valid: true},
		Name:               toNullString(params.Name),
		Description:        field.StringToNullString(params.Description),
		ClearDescription:   params.Description.IsClear(),
		IsCommissionExempt: commissionPolicyToNullBool(params.CommissionPolicy),
		IsFreightExempt:    freightPolicyToNullBool(params.FreightPolicy),
		UnitGroupID:        toNullString(params.UnitGroupID),
		SupplyCode:         supply,
		ClearSupply:        params.Supply.IsClear(),
		IsActive:           isActive,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Product line not found."))
	}

	return r.Get(ctx, domain.GetProductLineParams{AccountID: params.AccountID, ProductLineID: params.ProductLineID})
}

// Delete removes the line along with the customer and account-group access granted to it.
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
		return tracing.Trace(span, apierror.NewNotFoundError("Product line not found."))
	}

	if err := r.queries.DeleteAccountRelationProductLinesForProductLine(ctx, params.ProductLineID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	if err := r.queries.DeleteAccountGroupProductLinesForProductLine(ctx, params.ProductLineID); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
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

func (r *productLineRepoImpl) FindByNames(ctx context.Context, accountID string, names []string) ([]*domain.ProductLineFull, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.find_by_names")
	defer span.End()

	if len(names) == 0 {
		return nil, nil
	}

	rows, err := r.queries.FindProductLinesByNames(ctx, sqlc.FindProductLinesByNamesParams{
		Names:     names,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.ProductLineFull, len(rows))
	for i, row := range rows {
		out[i] = mapProductLineRow(sqlc.GetProductLineRow(row))
	}
	return out, nil
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
		out[i] = mapProductLineRow(sqlc.GetProductLineRow(row))
	}
	// Stitch unit group data — always include base_unit and associated_units so the API gateway's SubField resolver has everything it needs.
	//
	// A group the tenant cannot see leaves the line's UnitGroup nil rather than failing the batch: this backs the list loader, so one line pointing at a group scoped to another account would otherwise 404 the entire page.
	unitGroupIDs := make([]string, 0, len(out))
	for _, pl := range out {
		if pl.UnitGroupID != "" {
			unitGroupIDs = append(unitGroupIDs, pl.UnitGroupID)
		}
	}
	unitGroups, apiErr := r.GetUnitGroups(ctx, accountID, unitGroupIDs, []string{"unit_group.base_unit", "unit_group.associated_units"})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, pl := range out {
		pl.UnitGroup = unitGroups[pl.UnitGroupID]
	}
	return out, nil
}

// GetUnitGroups is the batched form of GetUnitGroup, keyed by id: at most three queries however many
// groups are asked for. Groups the account cannot see are absent.
func (r *productLineRepoImpl) GetUnitGroups(ctx context.Context, accountID string, ids []string, includes []string) (map[string]*domain.ProductLineUnitGroup, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_unit_groups")
	defer span.End()

	groups := make(map[string]*domain.ProductLineUnitGroup, len(ids))
	if len(ids) == 0 {
		return groups, nil
	}

	rows, err := r.queries.GetUnitGroupsForProductLinesByIDs(ctx, sqlc.GetUnitGroupsForProductLinesByIDsParams{
		Ids:       ids,
		AccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(rows) == 0 {
		return groups, nil
	}

	groupIDs := make([]string, len(rows))
	baseUnitIDs := make([]string, 0, len(rows))
	for i, row := range rows {
		groups[row.ID] = &domain.ProductLineUnitGroup{
			ID:         row.ID,
			Name:       row.Name,
			BaseUnitID: row.BaseUnitID,
			Type:       row.UnitTypeCode,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		groupIDs[i] = row.ID
		if row.BaseUnitID != "" {
			baseUnitIDs = append(baseUnitIDs, row.BaseUnitID)
		}
	}

	if unitGroupIncludeCovers(includes, "base_unit") && len(baseUnitIDs) > 0 {
		unitRows, err := r.queries.GetUnitsByIDs(ctx, baseUnitIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		units := make(map[string]domain.LightUnit, len(unitRows))
		for _, row := range unitRows {
			units[row.ID] = mapGetUnitsByIDsRowToLightUnit(row)
		}
		for _, ug := range groups {
			if lu, ok := units[ug.BaseUnitID]; ok {
				ug.BaseUnit = &lu
			}
		}
	}

	if unitGroupIncludeCovers(includes, "associated_units") {
		ugUnitRows, err := r.queries.ListUnitGroupUnitsByUnitGroupIDs(ctx, groupIDs)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, ug := range groups {
			ug.AssociatedUnits = []*domain.UnitGroupUnit{}
		}
		for _, row := range ugUnitRows {
			if ug := groups[row.UnitGroupID]; ug != nil {
				ug.AssociatedUnits = append(ug.AssociatedUnits, mapUnitGroupUnitsByUnitGroupIDsRow(row))
			}
		}
	}
	return groups, nil
}

// unitGroupIncludeCovers reports whether includes ask for a unit group's sub-resource, addressed from
// the unit group or through a product line; includes may nest further (…associated_units.unit).
func unitGroupIncludeCovers(includes []string, subResource string) bool {
	for _, frag := range []string{"unit_group." + subResource, "product_line.unit_group." + subResource} {
		for _, inc := range includes {
			if inc == frag || strings.HasPrefix(inc, frag+".") {
				return true
			}
		}
	}
	return false
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
