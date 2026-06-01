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
		ObjectType: constants.ObjectTypeAgentRun,
		Load:       resourceloaders.LoadAgentRuns,
		Subs: []resourcekit.SubField{
			{Key: "actions", Populate: populateActionsOnAgentRun},
			{
				Key:         "definition",
				Target:      constants.ObjectTypeAgentDefinition,
				ExtractRefs: extractDefinitionRefsFromAgentRun,
				Populate:    populateDefinitionOnAgentRun,
			},
			{Key: "steps", Populate: populateStepsOnAgentRun},
		},
	})
}

func populateActionsOnAgentRun(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.AgentRun)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeAgentRun, r.ID, "actions")
	if !ok {
		return
	}
	r.Actions = v.(*apiresource.List[apiresource.AgentAction])
}

func extractDefinitionRefsFromAgentRun(_ context.Context, parent any) []any {
	r := parent.(*apiresource.AgentRun)
	if r.Definition == nil {
		return nil
	}
	return []any{r.Definition}
}

func populateDefinitionOnAgentRun(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.AgentRun)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeAgentRun, r.ID, "definition")
	if !ok {
		return
	}
	r.Definition = v.(*apiresource.AgentDefinition)
}

func populateStepsOnAgentRun(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.AgentRun)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeAgentRun, r.ID, "steps")
	if !ok {
		return
	}
	r.Steps = v.(*apiresource.List[apiresource.AgentRunStep])
}
