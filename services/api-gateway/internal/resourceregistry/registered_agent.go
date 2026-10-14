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
		ObjectType: constants.ObjectTypeAgentDefinition,
		Load:       resourceloaders.LoadAgentDefinitions,
		Subs: []include.SubField{
			{Key: "config", Populate: populateConfigOnAgent},
			{Key: "tools", Populate: populateToolsOnAgent},
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromAgent,
				Populate:    populateRoleOnAgent,
			},
		},
	})
}

func populateConfigOnAgent(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentDefinition, a.ID, "config")
	if !ok {
		return
	}
	a.Config = v.(*apiresource.AgentDefinitionConfig)
}

func populateToolsOnAgent(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentDefinition, a.ID, "tools")
	if !ok {
		return
	}
	a.Tools = v.(*apiresource.List[apiresource.AgentDefinitionTool])
}

func extractRoleIDFromAgent(ctx context.Context, parent any) []string {
	a := parent.(*apiresource.AgentDefinition)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAgentDefinition, a.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnAgent(ctx context.Context, parent any, loaded map[string]any) {
	a := parent.(*apiresource.AgentDefinition)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAgentDefinition, a.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		a.Role = v.(*apiresource.Role)
	}
}
