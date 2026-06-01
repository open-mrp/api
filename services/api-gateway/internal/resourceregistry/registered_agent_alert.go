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
		ObjectType: constants.ObjectTypeAgentAlert,
		Load:       resourceloaders.LoadAgentAlerts,
		Subs: []resourcekit.SubField{
			{Key: "run", Populate: populateRunOnAgentAlert},
			{Key: "action", Populate: populateActionOnAgentAlert},
		},
	})
}

func populateRunOnAgentAlert(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentAlert)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentAlert, a.ID, "run")
	if !ok {
		return
	}
	a.Run = v.(*apiresource.AgentRun)
}

func populateActionOnAgentAlert(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.AgentAlert)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAgentAlert, a.ID, "action")
	if !ok {
		return
	}
	a.Action = v.(*apiresource.AgentAction)
}
