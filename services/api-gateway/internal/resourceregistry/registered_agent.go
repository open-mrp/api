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
		ObjectType: constants.ObjectTypeAgentDefinition,
		Load:       resourceloaders.LoadAgentDefinitions,
		Subs: []resourcekit.SubField{
			{Key: "config", Populate: populateConfigOnAgent},
			{Key: "tools", Populate: populateToolsOnAgent},
			{Key: "role", Populate: populateRoleOnAgent},
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

func populateRoleOnAgent(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentDefinition, a.ID, "role")
	if !ok {
		return
	}
	a.Role = v.(*apiresource.Role)
}
