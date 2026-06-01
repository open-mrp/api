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
		ObjectType: constants.ObjectTypeAvailableTool,
		Load:       resourceloaders.LoadAvailableTools,
		Subs:       []resourcekit.SubField{},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeToolGroup,
		Load:       resourceloaders.LoadToolGroups,
		Subs: []resourcekit.SubField{
			{Key: "tools", Populate: populateToolsOnToolGroup},
		},
	})
}

func populateToolsOnToolGroup(ctx context.Context, parent any, _ map[string]any) {
	tg := parent.(*apiresource.ToolGroup)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeToolGroup, tg.ID, "tools")
	if !ok {
		return
	}
	tg.Tools = v.(*apiresource.List[apiresource.AvailableTool])
}
