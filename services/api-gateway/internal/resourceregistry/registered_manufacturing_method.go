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
		ObjectType: constants.ObjectTypeManufacturingMethod,
		Load:       resourceloaders.LoadManufacturingMethods,
		Subs: []include.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  metaIDs(constants.ObjectTypeManufacturingMethod, func(p any) string { return p.(*apiresource.ManufacturingMethod).ID }, "item_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					m := parent.(*apiresource.ManufacturingMethod)
					id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeManufacturingMethod, m.ID, "item_id")
					if v, ok := loaded[id]; ok {
						m.Item = v.(*apiresource.Item)
					}
				},
			},
			// Every method is a standard method for now, so no method names an order line.
			{Key: "sales_order_line", Populate: func(context.Context, any, map[string]any) {}},
			{
				Key:         "operations",
				Target:      constants.ObjectTypeMethodOperation,
				Cardinality: include.CardinalityList,
				ExtractIDs:  metaIDs(constants.ObjectTypeManufacturingMethod, func(p any) string { return p.(*apiresource.ManufacturingMethod).ID }, resourceloaders.MethodOperationIDsKey),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					m := parent.(*apiresource.ManufacturingMethod)
					items := metaMembers[apiresource.MethodOperation](ctx, constants.ObjectTypeManufacturingMethod, m.ID, resourceloaders.MethodOperationIDsKey, loaded)
					m.Operations = apiresource.NewEmbeddedList(items, false, resourceloaders.ManufacturingMethodMembersPath(m.ID, "operations"),
						metaCursor[apiresource.MethodOperation](ctx, constants.ObjectTypeManufacturingMethod, m.ID, resourceloaders.MethodOperationsCursorKey))
				},
			},
			{
				Key:         "materials",
				Target:      constants.ObjectTypeMethodMaterial,
				Cardinality: include.CardinalityList,
				ExtractIDs:  metaIDs(constants.ObjectTypeManufacturingMethod, func(p any) string { return p.(*apiresource.ManufacturingMethod).ID }, resourceloaders.MethodMaterialIDsKey),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					m := parent.(*apiresource.ManufacturingMethod)
					items := metaMembers[apiresource.MethodMaterial](ctx, constants.ObjectTypeManufacturingMethod, m.ID, resourceloaders.MethodMaterialIDsKey, loaded)
					m.Materials = apiresource.NewEmbeddedList(items, false, resourceloaders.ManufacturingMethodMembersPath(m.ID, "materials"),
						metaCursor[apiresource.MethodMaterial](ctx, constants.ObjectTypeManufacturingMethod, m.ID, resourceloaders.MethodMaterialsCursorKey))
				},
			},
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeMethodOperation,
		Load:       resourceloaders.LoadMethodOperations,
		Subs: []include.SubField{
			{
				Key:         "work_center",
				Target:      constants.ObjectTypeWorkCenter,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  metaIDs(constants.ObjectTypeMethodOperation, func(p any) string { return p.(*apiresource.MethodOperation).ID }, "work_center_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					op := parent.(*apiresource.MethodOperation)
					id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeMethodOperation, op.ID, "work_center_id")
					if v, ok := loaded[id]; ok {
						op.WorkCenter = v.(*apiresource.WorkCenter)
					}
				},
			},
			{
				Key:         "machines",
				Target:      constants.ObjectTypeMachine,
				Cardinality: include.CardinalityList,
				ExtractIDs:  metaIDs(constants.ObjectTypeMethodOperation, func(p any) string { return p.(*apiresource.MethodOperation).ID }, "machine_ids"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					op := parent.(*apiresource.MethodOperation)
					items := metaMembers[apiresource.Machine](ctx, constants.ObjectTypeMethodOperation, op.ID, "machine_ids", loaded)
					op.Machines = apiresource.NewList(items, apiresource.PageInfo{})
				},
			},
			{
				Key:         "scanning_station",
				Target:      constants.ObjectTypeScanningStation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  metaIDs(constants.ObjectTypeMethodOperation, func(p any) string { return p.(*apiresource.MethodOperation).ID }, "scanning_station_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					op := parent.(*apiresource.MethodOperation)
					id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeMethodOperation, op.ID, "scanning_station_id")
					if v, ok := loaded[id]; ok {
						op.ScanningStation = v.(*apiresource.ScanningStation)
					}
				},
			},
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeMethodMaterial,
		Load:       resourceloaders.LoadMethodMaterials,
		Subs: []include.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  metaIDs(constants.ObjectTypeMethodMaterial, func(p any) string { return p.(*apiresource.MethodMaterial).ID }, "item_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					m := parent.(*apiresource.MethodMaterial)
					id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeMethodMaterial, m.ID, "item_id")
					if v, ok := loaded[id]; ok {
						m.Item = v.(*apiresource.Item)
					}
				},
			},
			{
				Key:         "operation",
				Target:      constants.ObjectTypeMethodOperation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  metaIDs(constants.ObjectTypeMethodMaterial, func(p any) string { return p.(*apiresource.MethodMaterial).ID }, "operation_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					m := parent.(*apiresource.MethodMaterial)
					id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeMethodMaterial, m.ID, "operation_id")
					if v, ok := loaded[id]; ok {
						m.Operation = v.(*apiresource.MethodOperation)
					}
				},
			},
		},
	})
}

// metaIDs extracts the IDs a parent's load meta holds under key: one string, or a list.
func metaIDs(parentType constants.ObjectType, parentID func(any) string, key string) func(ctx context.Context, parent any) []string {
	return func(ctx context.Context, parent any) []string {
		meta := include.GetLoadMeta(ctx)
		id := parentID(parent)
		if ids, ok := meta.GetStrings(parentType, id, key); ok {
			return ids
		}
		if v, _ := meta.GetString(parentType, id, key); v != "" {
			return []string{v}
		}
		return nil
	}
}

// metaMembers is the loaded members a parent's load meta lists under key, in its order.
func metaMembers[T any](ctx context.Context, parentType constants.ObjectType, parentID, key string, loaded map[string]any) []T {
	ids, _ := include.GetLoadMeta(ctx).GetStrings(parentType, parentID, key)
	items := make([]T, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*T)))
		}
	}
	return items
}
