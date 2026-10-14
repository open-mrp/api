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
				Key:         "department",
				Target:      constants.ObjectTypeDepartment,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractDepartmentIDFromMachine,
				Populate:    populateDepartmentOnMachine,
			},
		},
	})
}

func extractDepartmentIDFromMachine(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Machine)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachine, m.ID, "department_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateDepartmentOnMachine(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Machine)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachine, m.ID, "department_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.Department = v.(*apiresource.Department)
	}
}
