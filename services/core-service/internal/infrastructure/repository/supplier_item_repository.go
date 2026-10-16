package repository

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

func supplierItemCreatedAt(s *domain.SupplierItem) time.Time { return s.CreatedAt }
func supplierItemID(s *domain.SupplierItem) string           { return s.ID }

// supplierItemListPageQuery selects one page of a supplier's item IDs in list order, emitting only the predicates the
// request uses. ListSupplierItemsByIDs reads the page's rows.
func supplierItemListPageQuery(params domain.ListSupplierItemsParams, cursor *pagination.StringCursor, limit int32) (string, []any) {
	var f listFilter
	f.add("sm.owner_account_id = ?", params.OwnerAccountID)
	f.add("sm.supplier_account_id = ?", params.SupplierAccountID)
	if params.Query != nil && *params.Query != "" {
		like := "%" + db.EscapeLike(*params.Query) + "%"
		f.add("(sm.supplier_part_number LIKE ? OR sm.supplier_description LIKE ?)", like, like)
	}
	if params.IsActive != nil {
		f.add("sm.is_active = ?", *params.IsActive)
	}
	if len(params.ItemIDs) > 0 {
		f.add("sm.material_id IN (SELECT m.id FROM material m WHERE m.item_id IN ("+placeholders(len(params.ItemIDs))+"))",
			stringArgs(params.ItemIDs)...)
	}
	f.createdBetween("sm.created_at", params.Created)

	order := f.keyset(cursor, "sm.created_at", "sm.id")
	return "SELECT sm.id FROM supplier_material sm FORCE INDEX (supplier_material_owner_supplier_created_idx)" +
		f.whereSQL() + "\nORDER BY " + order + "\nLIMIT ?", append(f.args, limit)
}

func mapSupplierItemRow(row sqlc.ListSupplierItemsByIDsRow) *domain.SupplierItem {
	return &domain.SupplierItem{
		ID:                  row.ID,
		SupplierAccountID:   row.SupplierAccountID,
		ItemID:              row.ItemID,
		MaterialID:          row.MaterialID,
		SupplierSKU:         row.SupplierPartNumber,
		SupplierDescription: nullStringPtr(row.SupplierDescription),
		Status:              constants.ArchiveStatusFromIsActive(row.IsActive),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func (r *supplierMaterialRepoImpl) readItems(ctx context.Context, ownerAccountID, supplierAccountID string, ids []string) (map[string]*domain.SupplierItem, *apierror.APIError) {
	byID := make(map[string]*domain.SupplierItem, len(ids))
	if len(ids) == 0 {
		return byID, nil
	}
	rows, err := r.queries.ListSupplierItemsByIDs(ctx, sqlc.ListSupplierItemsByIDsParams{
		OwnerAccountID:    ownerAccountID,
		SupplierAccountID: supplierAccountID,
		Ids:               ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		byID[row.ID] = mapSupplierItemRow(row)
	}
	return byID, nil
}

func (r *supplierMaterialRepoImpl) ListItems(ctx context.Context, params domain.ListSupplierItemsParams) (*domain.ListSupplierItemsResult, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_item.list")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	query, args := supplierItemListPageQuery(params, cursor, params.Limit+1)
	ids, err := selectStrings(ctx, r.queries.DB(), query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	byID, apiErr := r.readItems(ctx, params.OwnerAccountID, params.SupplierAccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	items, pageInfo := pagination.BuildPageString(inPageOrder(ids, byID), params.Limit, cursorDirection(cursor), supplierItemCreatedAt, supplierItemID)
	return &domain.ListSupplierItemsResult{SupplierItems: items, PageInfo: pageInfo}, nil
}

func (r *supplierMaterialRepoImpl) GetItem(ctx context.Context, ownerAccountID, supplierAccountID, id string) (*domain.SupplierItem, *apierror.APIError) {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_item.get")
	defer span.End()

	byID, apiErr := r.readItems(ctx, ownerAccountID, supplierAccountID, []string{id})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	item, ok := byID[id]
	if !ok {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("No supplier item found with the provided ID.").WithParam("id"))
	}
	return item, nil
}

func (r *supplierMaterialRepoImpl) UpdateItem(ctx context.Context, ownerAccountID string, item *domain.SupplierItem) *apierror.APIError {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_item.update")
	defer span.End()

	err := r.queries.UpdateSupplierItem(ctx, sqlc.UpdateSupplierItemParams{
		SupplierPartNumber:  item.SupplierSKU,
		SupplierDescription: toNullString(item.SupplierDescription),
		IsActive:            item.Status.IsActive(),
		ID:                  item.ID,
		OwnerAccountID:      ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *supplierMaterialRepoImpl) DeleteItem(ctx context.Context, ownerAccountID, id string) *apierror.APIError {
	ctx, span := supplierMaterialRepoTracer.Start(ctx, "repository.supplier_item.delete")
	defer span.End()

	_, err := r.queries.DeleteSupplierMaterial(ctx, sqlc.DeleteSupplierMaterialParams{ID: id, OwnerAccountID: ownerAccountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
