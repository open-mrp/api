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
		ObjectType: constants.ObjectTypeAgentDefinition,
		Load:       resourceloaders.LoadAgentDefinitions,
		Subs: []resourcekit.SubField{
			{Key: "config", Populate: populateConfigOnAgent},
			{Key: "tools", Populate: populateToolsOnAgent},
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromAgent,
				Populate:    populateRoleOnAgent,
			},
		},
	})
}

func populateConfigOnAgent(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentDefinition, a.ID, "config")
	if !ok {
		return
	}
	a.Config = v.(*apiresource.AgentDefinitionConfig)
}

func populateToolsOnAgent(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentDefinition, a.ID, "tools")
	if !ok {
		return
	}
	a.Tools = v.(*apiresource.List[apiresource.AgentDefinitionTool])
}

func extractRoleIDFromAgent(ctx context.Context, parent any) []string {
	a := parent.(*apiresource.AgentDefinition)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAgentDefinition, a.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnAgent(ctx context.Context, parent any, loaded map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAgentDefinition, a.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		a.Role = v.(*apiresource.Role)
	}
}
