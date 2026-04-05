package agenttoolep

import (
	"encoding/json"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/agent"
)

func AvailableToolPresenter(t *pb.AvailableToolInfo) apiresource.AvailableTool {
	if t == nil {
		return apiresource.AvailableTool{}
	}

	perms := t.RequiredPermissions
	if perms == nil {
		perms = []string{}
	}

	return apiresource.AvailableTool{
		ID:                  t.Id,
		Object:              constants.ObjectTypeAvailableTool,
		Name:                t.DisplayName,
		Description:         &t.Description,
		ConfigSchema:        json.RawMessage(t.ConfigSchemaJson),
		Category:            t.Category,
		RequiredPermissions: perms,
	}
}

func AvailableToolListPresenter(resp *pb.ListAvailableToolsResponse) *apiresource.List[apiresource.AvailableTool] {
	if resp == nil {
		return apiresource.NewList[apiresource.AvailableTool](nil, apiresource.PageInfo{})
	}

	tools := make([]apiresource.AvailableTool, len(resp.Tools))
	for i, t := range resp.Tools {
		tools[i] = AvailableToolPresenter(t)
	}

	return apiresource.NewList(tools, apiresource.PageInfo{})
}

func ToolGroupPresenter(g *pb.ToolGroupInfo, tools []apiresource.AvailableTool) apiresource.ToolGroup {
	if g == nil {
		return apiresource.ToolGroup{}
	}

	return apiresource.ToolGroup{
		ID:          g.Id,
		Object:      constants.ObjectTypeToolGroup,
		Name:        g.Name,
		Description: g.Description,
		Slug:        g.Slug,
		Icon:        g.Icon,
		SortOrder:   g.SortOrder,
		Tools:       apiresource.NewList(tools, apiresource.PageInfo{}),
	}
}

func ToolGroupListPresenter(resp *pb.ListAvailableToolsResponse, includeKeys []string) *apiresource.List[apiresource.ToolGroup] {
	if resp == nil {
		return apiresource.NewList[apiresource.ToolGroup](nil, apiresource.PageInfo{})
	}

	includes := make(map[string]bool, len(includeKeys))
	for _, k := range includeKeys {
		includes[k] = true
	}

	// Build per-group tool lists if tools are included.
	var toolsByGroup map[string][]apiresource.AvailableTool
	if includes["tools"] {
		toolsByGroup = make(map[string][]apiresource.AvailableTool, len(resp.Groups))
		for _, t := range resp.Tools {
			toolsByGroup[t.GroupId] = append(toolsByGroup[t.GroupId], AvailableToolPresenter(t))
		}
	}

	groups := make([]apiresource.ToolGroup, len(resp.Groups))
	for i, g := range resp.Groups {
		var tools []apiresource.AvailableTool
		if toolsByGroup != nil {
			tools = toolsByGroup[g.Id]
			if tools == nil {
				tools = []apiresource.AvailableTool{}
			}
		}
		groups[i] = ToolGroupPresenter(g, tools)
	}

	return apiresource.NewList(groups, apiresource.PageInfo{})
}
