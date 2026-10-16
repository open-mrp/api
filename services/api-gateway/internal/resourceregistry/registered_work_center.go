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
		ObjectType: constants.ObjectTypeWorkCenter,
		Load:       resourceloaders.LoadWorkCenters,
		Subs: []include.SubField{
			{
				Key:         "location",
				Target:      constants.ObjectTypeLocation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  workCenterMetaIDs("location_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					wc := parent.(*apiresource.WorkCenter)
					id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeWorkCenter, wc.ID, "location_id")
					if v, ok := loaded[id]; ok {
						wc.Location = v.(*apiresource.Location)
					}
				},
			},
			{
				Key:         "scanning_stations",
				Target:      constants.ObjectTypeScanningStation,
				Cardinality: include.CardinalityList,
				ExtractIDs:  workCenterMetaIDs("scanning_station_ids"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					wc := parent.(*apiresource.WorkCenter)
					items := embeddedMembers[apiresource.ScanningStation](ctx, wc.ID, "scanning_station_ids", loaded)
					wc.ScanningStations = apiresource.NewEmbeddedList(items, false, resourceloaders.WorkCenterMembersListPath("scanning-stations", wc.ID),
						metaCursor[apiresource.ScanningStation](ctx, constants.ObjectTypeWorkCenter, wc.ID, resourceloaders.WorkCenterStationsCursorKey))
				},
			},
			{
				Key:         "machines",
				Target:      constants.ObjectTypeMachine,
				Cardinality: include.CardinalityList,
				ExtractIDs:  workCenterMetaIDs("machine_ids"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					wc := parent.(*apiresource.WorkCenter)
					items := embeddedMembers[apiresource.Machine](ctx, wc.ID, "machine_ids", loaded)
					wc.Machines = apiresource.NewEmbeddedList(items, false, resourceloaders.WorkCenterMembersListPath("machines", wc.ID),
						metaCursor[apiresource.Machine](ctx, constants.ObjectTypeWorkCenter, wc.ID, resourceloaders.WorkCenterMachinesCursorKey))
				},
			},
		},
	})
}

// workCenterMetaIDs extracts the IDs a work center's load meta holds under key: one string, or a list.
func workCenterMetaIDs(key string) func(ctx context.Context, parent any) []string {
	return func(ctx context.Context, parent any) []string {
		wc := parent.(*apiresource.WorkCenter)
		meta := include.GetLoadMeta(ctx)
		if ids, ok := meta.GetStrings(constants.ObjectTypeWorkCenter, wc.ID, key); ok {
			return ids
		}
		if id, _ := meta.GetString(constants.ObjectTypeWorkCenter, wc.ID, key); id != "" {
			return []string{id}
		}
		return nil
	}
}

// embeddedMembers is the loaded members a work center's load meta lists under key, in its order. The loader carries
// one more than an embedded list shows, so NewEmbeddedList notices a longer list.
func embeddedMembers[T any](ctx context.Context, workCenterID, key string, loaded map[string]any) []T {
	ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeWorkCenter, workCenterID, key)
	items := make([]T, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*T)))
		}
	}
	return items
}

// metaCursor is an embedded list's cursorAfter: the list cursor the parent's load meta holds under key, which the loader
// set after the last item the list embeds.
func metaCursor[T any](ctx context.Context, parentType constants.ObjectType, parentID, key string) func(T) string {
	return func(T) string {
		cursor, _ := include.GetLoadMeta(ctx).GetString(parentType, parentID, key)
		return cursor
	}
}
