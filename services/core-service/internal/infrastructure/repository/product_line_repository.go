package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
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
	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
		AccountID:        accountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
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
	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
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
	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
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

	err := r.queries.InsertProductLine(ctx, sqlc.InsertProductLineParams{
		ID:                 id,
		Name:               params.Name,
		IsCommissionExempt: params.CommissionPolicy.ToBool(),
		IsFreightExempt:    params.FreightPolicy.ToBool(),
		UnitGroupID:        params.UnitGroupID,
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

	result, err := r.queries.UpdateProductLine(ctx, sqlc.UpdateProductLineParams{
		ID:                 params.ProductLineID,
		AccountID:          gosql.NullString{String: params.AccountID, Valid: true},
		Name:               toNullString(params.Name),
		IsCommissionExempt: commissionPolicyToNullBool(params.CommissionPolicy),
		IsFreightExempt:    freightPolicyToNullBool(params.FreightPolicy),
		UnitGroupID:        toNullString(params.UnitGroupID),
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

func (r *productLineRepoImpl) GetUnitGroup(ctx context.Context, unitGroupID string, includes []string) (*domain.ProductLineUnitGroup, *apierror.APIError) {
	ctx, span := productLineRepoTracer.Start(ctx, "repository.product_line.get_unit_group")
	defer span.End()

	row, err := r.queries.GetUnitGroupForProductLine(ctx, unitGroupID)
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
	// Stitch unit group data — always include base_unit and associated_units
	// so the API gateway's SubField resolver has everything it needs.
	for _, pl := range out {
		ug, apiErr := r.GetUnitGroup(ctx, pl.UnitGroupID, []string{"unit_group.base_unit", "unit_group.associated_units"})
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
	return &domain.ProductLineFull{
		ID:               row.ID,
		Name:             row.Name,
		Description:      description,
		Notes:            notes,
		CommissionPolicy: constants.CommissionPolicyFromBool(row.IsCommissionExempt),
		FreightPolicy:    constants.FreightPolicyFromBool(row.IsFreightExempt),
		UnitGroupID:      row.UnitGroupID,
		AccountID:        accountID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
