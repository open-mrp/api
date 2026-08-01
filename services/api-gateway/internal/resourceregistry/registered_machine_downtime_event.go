package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeMachineDowntimeEvent,
		Load:       resourceloaders.LoadMachineDowntimeEvents,
		Subs: []resourcekit.SubField{
			{
				Key:         "machine",
				Target:      constants.ObjectTypeMachine,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDowntimeRefIDs("machine_id"),
				Populate:    populateDowntimeMachine,
			},
			{
				Key:         "department",
				Target:      constants.ObjectTypeDepartment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDowntimeRefIDs("department_id"),
				Populate:    populateDowntimeDepartment,
			},
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDowntimeRefIDs("item_id"),
				Populate:    populateDowntimeItem,
			},
			{
				Key:         "reported_by",
				Target:      constants.ObjectTypeActor,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDowntimeRefIDs("reported_by_id"),
				Populate:    populateDowntimeReportedBy,
			},
		},
	})
}

func machineDowntimeEventID(parent any) string {
	if e, ok := parent.(*apiresource.MachineDowntimeEvent); ok {
		return e.ID
	}
	return ""
}

// downtimeRefID reads one stashed FK id off LoadMeta. Every expandable on this resource is a plain single-id reference, so extraction and population differ only by which key they read.
func downtimeRefID(ctx context.Context, parent any, key string) string {
	eventID := machineDowntimeEventID(parent)
	if eventID == "" {
		return ""
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachineDowntimeEvent, eventID, key)
	return id
}

func extractDowntimeRefIDs(key string) func(context.Context, any) []string {
	return func(ctx context.Context, parent any) []string {
		id := downtimeRefID(ctx, parent, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}

func populateDowntimeMachine(ctx context.Context, parent any, loaded map[string]any) {
	e, ok := parent.(*apiresource.MachineDowntimeEvent)
	if !ok {
		return
	}
	if v, ok := loaded[downtimeRefID(ctx, parent, "machine_id")]; ok {
		e.Machine = v.(*apiresource.Machine)
	}
}

func populateDowntimeDepartment(ctx context.Context, parent any, loaded map[string]any) {
	e, ok := parent.(*apiresource.MachineDowntimeEvent)
	if !ok {
		return
	}
	if v, ok := loaded[downtimeRefID(ctx, parent, "department_id")]; ok {
		e.Department = v.(*apiresource.Department)
	}
}

func populateDowntimeItem(ctx context.Context, parent any, loaded map[string]any) {
	e, ok := parent.(*apiresource.MachineDowntimeEvent)
	if !ok {
		return
	}
	if v, ok := loaded[downtimeRefID(ctx, parent, "item_id")]; ok {
		e.Item = v.(*apiresource.Item)
	}
}

func populateDowntimeReportedBy(ctx context.Context, parent any, loaded map[string]any) {
	e, ok := parent.(*apiresource.MachineDowntimeEvent)
	if !ok {
		return
	}
	if v, ok := loaded[downtimeRefID(ctx, parent, "reported_by_id")]; ok {
		e.ReportedBy = v.(*apiresource.Actor)
	}
}
