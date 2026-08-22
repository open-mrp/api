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
			{Key: "item", Populate: populateItemOnInventoryChangeLog},
			{Key: "quantity", Populate: populateQuantityOnInventoryChangeLog},
			{Key: "responsible_user", Populate: populateResponsibleUserOnInventoryChangeLog},
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

func populateItemOnInventoryChangeLog(ctx context.Context, parent any, _ map[string]any) {
	icl := parent.(*apiresource.InventoryChangeLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeInventoryChangeLog, icl.ID, "item")
	if !ok {
		return
	}
	icl.Item = v.(*apiresource.Item)
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

func populateResponsibleUserOnInventoryChangeLog(ctx context.Context, parent any, _ map[string]any) {
	icl := parent.(*apiresource.InventoryChangeLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeInventoryChangeLog, icl.ID, "responsible_user")
	if !ok {
		return
	}
	icl.ResponsibleUser = v.(*apiresource.User)
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
