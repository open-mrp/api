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
		ObjectType: constants.ObjectTypeAvailableTool,
		Load:       resourceloaders.LoadAvailableTools,
		Subs:       []include.SubField{},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeToolGroup,
		Load:       resourceloaders.LoadToolGroups,
		Subs: []include.SubField{
			{Key: "tools", Populate: populateToolsOnToolGroup},
		},
	})
}

func populateToolsOnToolGroup(ctx context.Context, parent any, _ map[string]any) {
	tg := parent.(*apiresource.ToolGroup)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeToolGroup, tg.ID, "tools")
	if !ok {
		return
	}
	tg.Tools = v.(*apiresource.List[apiresource.AvailableTool])
}
