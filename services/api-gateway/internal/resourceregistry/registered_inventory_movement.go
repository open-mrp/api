package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeInventoryMovement,
		Load:       resourceloaders.LoadInventoryChangeLogs,
		Subs: []include.SubField{
			movementRefSub("item", constants.ObjectTypeItem, "item_id", func(m *apiresource.InventoryMovement, v any) {
				m.Item = v.(*apiresource.Item)
			}),
			movementRefSub("location", constants.ObjectTypeLocation, "location_id", func(m *apiresource.InventoryMovement, v any) {
				m.Location = v.(*apiresource.Location)
			}),
			{
				// Carried inline: the movement query returns the lot's number alongside its id.
				Key: "lot",
				Populate: func(ctx context.Context, parent any, _ map[string]any) {
					m := parent.(*apiresource.InventoryMovement)
					if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeInventoryMovement, m.ID, "lot"); ok {
						m.Lot = v.(*apiresource.Lot)
					}
				},
			},
			messagingRecordSub("related", constants.ObjectTypeInventoryMovement,
				func(p any) string { return p.(*apiresource.InventoryMovement).ID }, "related",
				func(p any, r *apiresource.Record) { p.(*apiresource.InventoryMovement).Related = r }),
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeInventoryLevel,
		Load:       resourceloaders.LoadInventoryChangeLogs,
		Subs: []include.SubField{
			{
				Key: "item", Target: constants.ObjectTypeItem, Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, parent any) []string {
					id, _ := parent.(*apiresource.InventoryLevel).Refs()
					return nonEmpty(id)
				},
				Populate: func(_ context.Context, parent any, loaded map[string]any) {
					l := parent.(*apiresource.InventoryLevel)
					id, _ := l.Refs()
					if v, ok := loaded[id]; ok {
						l.Item = v.(*apiresource.Item)
					}
				},
			},
			{
				Key: "location", Target: constants.ObjectTypeLocation, Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, parent any) []string {
					_, id := parent.(*apiresource.InventoryLevel).Refs()
					return nonEmpty(id)
				},
				Populate: func(_ context.Context, parent any, loaded map[string]any) {
					l := parent.(*apiresource.InventoryLevel)
					_, id := l.Refs()
					if v, ok := loaded[id]; ok {
						l.Location = v.(*apiresource.Location)
					}
				},
			},
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeLotSizeDefault,
		Load:       resourceloaders.LoadInventoryChangeLogs,
		Subs: []include.SubField{
			{
				Key: "item", Target: constants.ObjectTypeItem, Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, parent any) []string {
					id, _ := parent.(*apiresource.LotSizeDefault).Refs()
					return nonEmpty(id)
				},
				Populate: func(_ context.Context, parent any, loaded map[string]any) {
					d := parent.(*apiresource.LotSizeDefault)
					id, _ := d.Refs()
					if v, ok := loaded[id]; ok {
						d.Item = v.(*apiresource.Item)
					}
				},
			},
			{
				Key: "item_category", Target: constants.ObjectTypeItemCategory, Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(_ context.Context, parent any) []string {
					_, id := parent.(*apiresource.LotSizeDefault).Refs()
					return nonEmpty(id)
				},
				Populate: func(_ context.Context, parent any, loaded map[string]any) {
					d := parent.(*apiresource.LotSizeDefault)
					_, id := d.Refs()
					if v, ok := loaded[id]; ok {
						d.ItemCategory = v.(*apiresource.ItemCategory)
					}
				},
			},
		},
	})
}

// movementRefSub includes a single resource a movement names by the ID its presenter stashed under metaKey.
func movementRefSub(key string, target constants.ObjectType, metaKey string, set func(*apiresource.InventoryMovement, any)) include.SubField {
	id := func(ctx context.Context, parent any) string {
		v, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeInventoryMovement, parent.(*apiresource.InventoryMovement).ID, metaKey)
		return v
	}
	return include.SubField{
		Key: key, Target: target, Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string { return nonEmpty(id(ctx, parent)) },
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			if v, ok := loaded[id(ctx, parent)]; ok {
				set(parent.(*apiresource.InventoryMovement), v)
			}
		},
	}
}

func nonEmpty(id string) []string {
	if id == "" {
		return nil
	}
	return []string{id}
}
