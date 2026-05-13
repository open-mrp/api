package agentep

import (
	"encoding/json"
	"sort"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
)

func unmarshalConfig(configJSON string) *apiresource.AgentDefinitionConfig {
	if configJSON == "" {
		return nil
	}
	var cfg apiresource.AgentDefinitionConfig
	_ = json.Unmarshal([]byte(configJSON), &cfg)
	cfg.Object = constants.ObjectTypeAgentDefinitionConfig
	if cfg.TriggerConfig != nil {
		cfg.TriggerConfig.Object = constants.ObjectTypeTriggerConfig
	}
	return &cfg
}

func AgentDefinitionPresenter(a *pb.AgentDefinitionInfo, roleInfo *ResolvedRole) apiresource.AgentDefinition {
	if a == nil {
		return apiresource.AgentDefinition{}
	}

	toolItems := make([]apiresource.AgentDefinitionTool, len(a.Tools))
	for i, t := range a.Tools {
		toolItems[i] = apiresource.AgentDefinitionTool{
			ID:     t.Id,
			Object: constants.ObjectTypeAgentDefinitionTool,
			Tool: apiresource.AvailableTool{
				ID:                  t.ToolId,
				Object:              constants.ObjectTypeAvailableTool,
				Name:                t.DisplayName,
				Description:         &t.Description,
				ConfigSchema:        json.RawMessage(t.ConfigSchemaJson),
				Category:            t.Category,
				RequiredPermissions: orEmptyStrSlice(t.RequiredPermissions),
			},
			Config:        json.RawMessage(t.ConfigJson),
			SortOrder:     t.SortOrder,
			RequireReview: t.RequireReview,
		}
	}
	tools := apiresource.NewList(toolItems, apiresource.PageInfo{})

	var role *apiresource.Role
	if a.RoleId != "" && roleInfo != nil {
		role = &apiresource.Role{
			ID:       a.RoleId,
			Object:   constants.ObjectTypeRole,
			Name:     roleInfo.Name,
			TypeCode: constants.RoleType(roleInfo.RoleType),
			Owner:    apiresource.SystemOwner(),
		}
		if roleInfo.Permissions != nil {
			perms := make([]string, 0, len(roleInfo.Permissions))
			for p := range roleInfo.Permissions {
				perms = append(perms, p)
			}
			sort.Strings(perms)
			role.Permissions = &perms
		}
	}

	accountStatus := constants.AgentAccountStatusInactive
	if a.AccountStatus != nil {
		accountStatus = constants.AgentAccountStatus(a.AccountStatus.StatusCode)
	}

	return apiresource.AgentDefinition{
		ID:             a.Id,
		Object:         constants.ObjectTypeAgentDefinition,
		Name:           a.Name,
		Slug:           a.Slug,
		Description:    &a.Description,
		DefinitionType: constants.AgentDefinitionType(a.DefinitionType),
		CategoryCode:   a.CategoryCode,
		TriggerType:    constants.AgentTriggerType(a.TriggerType),
		IsEditable:     a.IsEditable,
		Role:           role,
		Config:         unmarshalConfig(a.ConfigJson),
		Tools:          tools,
		AccountStatus:  accountStatus,
		CreatedAt:      timeutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:      timeutil.TimestampToTime(a.UpdatedAt),
	}
}

func AgentDefinitionListPresenter(resp *pb.ListAgentDefinitionsResponse, roleResolver func(roleID string) *ResolvedRole) *apiresource.List[apiresource.AgentDefinition] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentDefinition](nil, apiresource.PageInfo{})
	}

	agents := make([]apiresource.AgentDefinition, len(resp.Agents))
	for i, a := range resp.Agents {
		var roleInfo *ResolvedRole
		if roleResolver != nil {
			roleInfo = roleResolver(a.RoleId)
		}
		agents[i] = AgentDefinitionPresenter(a, roleInfo)
	}

	return apiresource.NewList(agents, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func AgentTokenUsagePresenter(u *pb.AgentTokenUsageInfo) apiresource.AgentTokenUsage {
	if u == nil {
		return apiresource.AgentTokenUsage{}
	}

	return apiresource.AgentTokenUsage{
		ID:           u.Id,
		Object:       constants.ObjectTypeAgentTokenUsage,
		Date:         u.Date,
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalCost:    u.TotalCost,
		RunCount:     u.RunCount,
		CreatedAt:    timeutil.TimestampToTime(u.CreatedAt),
		UpdatedAt:    timeutil.TimestampToTime(u.UpdatedAt),
	}
}

func AgentTokenUsageListPresenter(resp *pb.ListTokenUsageResponse) *apiresource.List[apiresource.AgentTokenUsage] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentTokenUsage](nil, apiresource.PageInfo{})
	}

	usage := make([]apiresource.AgentTokenUsage, len(resp.Usage))
	for i, u := range resp.Usage {
		usage[i] = AgentTokenUsagePresenter(u)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = apiresource.PageInfo{
			NextCursor:  resp.PageInfo.NextCursor,
			PrevCursor:  resp.PageInfo.PrevCursor,
			HasNextPage: resp.PageInfo.HasNextPage,
			HasPrevPage: resp.PageInfo.HasPrevPage,
		}
	}

	return apiresource.NewList(usage, pageInfo)
}

func orEmptyStrSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
