package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInventoryChangeLog,
		Load:       resourceloaders.LoadInventoryChangeLogs,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromInventoryChangeLog,
				Populate:    populateItemOnInventoryChangeLog,
			},
			{Key: "quantity", Populate: populateQuantityOnInventoryChangeLog},
			{
				Key:         "responsible_user",
				Target:      constants.ObjectTypeUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractResponsibleUserIDFromInventoryChangeLog,
				Populate:    populateResponsibleUserOnInventoryChangeLog,
			},
			{
				Key:         "responsible_scanning_station",
				Target:      constants.ObjectTypeScanningStation,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractScanningStationIDFromInventoryChangeLog,
				Populate:    populateResponsibleScanningStationOnInventoryChangeLog,
			},
		},
	})
}

// itemIDOnInventoryChangeLog reads the affected item's id off LoadMeta so the item is hydrated through the shared batched item loader, which carries every base field (description, notes) and the item's own expandables — rather than a thin stub synthesized from the change-log join.
func itemIDOnInventoryChangeLog(ctx context.Context, parent any) string {
	icl := parent.(*apiresource.InventoryChangeLog)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeInventoryChangeLog, icl.ID, "item_id")
	return id
}

func extractItemIDFromInventoryChangeLog(ctx context.Context, parent any) []string {
	id := itemIDOnInventoryChangeLog(ctx, parent)
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnInventoryChangeLog(ctx context.Context, parent any, loaded map[string]any) {
	icl := parent.(*apiresource.InventoryChangeLog)
	if v, ok := loaded[itemIDOnInventoryChangeLog(ctx, parent)]; ok {
		icl.Item = v.(*apiresource.Item)
	}
}

func populateQuantityOnInventoryChangeLog(ctx context.Context, parent any, _ map[string]any) {
	icl := parent.(*apiresource.InventoryChangeLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeInventoryChangeLog, icl.ID, "quantity")
	if !ok {
		return
	}
	icl.Quantity = v.(*apiresource.Quantity)
}

// responsibleUserIDOnInventoryChangeLog reads the recording user's id off LoadMeta so the user is hydrated through the shared batched user loader, which carries every base field (email, username, image url) — rather than a stub built from the change-log join.
func responsibleUserIDOnInventoryChangeLog(ctx context.Context, parent any) string {
	icl := parent.(*apiresource.InventoryChangeLog)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeInventoryChangeLog, icl.ID, "responsible_user_id")
	return id
}

func extractResponsibleUserIDFromInventoryChangeLog(ctx context.Context, parent any) []string {
	id := responsibleUserIDOnInventoryChangeLog(ctx, parent)
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateResponsibleUserOnInventoryChangeLog(ctx context.Context, parent any, loaded map[string]any) {
	icl := parent.(*apiresource.InventoryChangeLog)
	if v, ok := loaded[responsibleUserIDOnInventoryChangeLog(ctx, parent)]; ok {
		icl.ResponsibleUser = v.(*apiresource.User)
	}
}

func extractScanningStationIDFromInventoryChangeLog(ctx context.Context, parent any) []string {
	icl := parent.(*apiresource.InventoryChangeLog)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeInventoryChangeLog, icl.ID, "responsible_scanning_station_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateResponsibleScanningStationOnInventoryChangeLog(ctx context.Context, parent any, loaded map[string]any) {
	icl := parent.(*apiresource.InventoryChangeLog)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeInventoryChangeLog, icl.ID, "responsible_scanning_station_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		icl.ResponsibleScanningStation = v.(*apiresource.ScanningStation)
	}
}
