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
		ObjectType: constants.ObjectTypeMachine,
		Load:       resourceloaders.LoadMachines,
		Subs: []resourcekit.SubField{
			{
				Key:         "department",
				Target:      constants.ObjectTypeDepartment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDepartmentIDFromMachine,
				Populate:    populateDepartmentOnMachine,
			},
		},
	})
}

func extractDepartmentIDFromMachine(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Machine)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachine, m.ID, "department_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateDepartmentOnMachine(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Machine)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachine, m.ID, "department_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.Department = v.(*apiresource.Department)
	}
}
