package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
)

// enrichItemCategoryProperties loads _item_categories_properties for all distinct item_category_id values present on the given items and attaches them to each domain.ItemCategory.
func enrichItemCategoryProperties(ctx context.Context, queries *sqlc.Queries, items []*domain.Item) *apierror.APIError {
	idsMap := make(map[string]struct{})
	for _, item := range items {
		if item != nil && item.Category != nil && item.Category.ID != "" {
			idsMap[item.Category.ID] = struct{}{}
		}
	}
	if len(idsMap) == 0 {
		return nil
	}
	ids := make([]string, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}
	rows, err := queries.ListItemCategoryPropertiesForCategories(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	byCat := make(map[string][]domain.ItemCategoryProperty)
	for _, row := range rows {
		byCat[row.ItemCategoryID] = append(byCat[row.ItemCategoryID], domain.ItemCategoryProperty{
			ID:   row.PropertyID,
			Name: row.PropertyName,
		})
	}
	for _, item := range items {
		if item != nil && item.Category != nil {
			if props, ok := byCat[item.Category.ID]; ok {
				item.Category.Properties = props
			}
		}
	}
	return nil
}

func itemsFromMaterials(ms []*domain.Material) []*domain.Item {
	out := make([]*domain.Item, 0, len(ms))
	for _, m := range ms {
		if m != nil && m.Item != nil {
			out = append(out, m.Item)
		}
	}
	return out
}

func itemsFromProductFulls(ps []*domain.ProductFull) []*domain.Item {
	out := make([]*domain.Item, 0, len(ps))
	for _, p := range ps {
		if p != nil && p.Item != nil {
			out = append(out, p.Item)
		}
	}
	return out
}

func itemsFromParts(ps []*domain.Part) []*domain.Item {
	out := make([]*domain.Item, 0, len(ps))
	for _, p := range ps {
		if p != nil && p.Item != nil {
			out = append(out, p.Item)
		}
	}
	return out
}
