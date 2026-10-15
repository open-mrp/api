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
		ObjectType: constants.ObjectTypeMachine,
		Load:       resourceloaders.LoadMachines,
		Subs: []include.SubField{
			{
				Key:         "work_center",
				Target:      constants.ObjectTypeWorkCenter,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractWorkCenterIDFromMachine,
				Populate:    populateWorkCenterOnMachine,
			},
		},
	})
}

func extractWorkCenterIDFromMachine(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Machine)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachine, m.ID, "work_center_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateWorkCenterOnMachine(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Machine)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachine, m.ID, "work_center_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.WorkCenter = v.(*apiresource.WorkCenter)
	}
}
