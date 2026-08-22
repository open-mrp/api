package agentep

import (
	"context"
	"encoding/json"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/agent"
	"github.com/open-mrp/api/shared/timeutil"
)

func unmarshalConfig(configJSON string) *apiresource.AgentDefinitionConfig {
	if configJSON == "" {
		return nil
	}
	// Read the persisted config into an intermediate wire shape and translate
	// into the public resource. Provider is derived at runtime and not surfaced.
	var wire struct {
		SystemPrompt       *string                    `json:"system_prompt"`
		Tier               *string                    `json:"tier"`
		Temperature        *float64                   `json:"temperature"`
		TriggerConfig      *apiresource.TriggerConfig `json:"trigger_config"`
		EndpointToolSlugs  []string                   `json:"endpoint_tool_slugs"`
		EndpointToolReview map[string]bool            `json:"endpoint_tool_review"`
	}
	_ = json.Unmarshal([]byte(configJSON), &wire)
	cfg := &apiresource.AgentDefinitionConfig{
		Object:             constants.ObjectTypeAgentDefinitionConfig,
		SystemPrompt:       wire.SystemPrompt,
		Temperature:        wire.Temperature,
		TriggerConfig:      wire.TriggerConfig,
		EndpointToolSlugs:  wire.EndpointToolSlugs,
		EndpointToolReview: wire.EndpointToolReview,
	}
	if wire.Tier != nil {
		t := constants.ModelTier(*wire.Tier)
		cfg.Tier = &t
	}
	if cfg.TriggerConfig != nil {
		cfg.TriggerConfig.Object = constants.ObjectTypeTriggerConfig
	}
	return cfg
}

func AgentDefinitionPresenter(a *pb.AgentDefinitionInfo) apiresource.AgentDefinition {
	if a == nil {
		return apiresource.AgentDefinition{}
	}

	accountStatus := constants.AgentAccountStatusInactive
	if a.AccountStatus != nil {
		accountStatus = constants.AgentAccountStatus(a.AccountStatus.StatusCode)
	}

	return apiresource.AgentDefinition{
		ID:             a.Id,
		Object:         constants.ObjectTypeAgentDefinition,
		DefinitionType: constants.AgentDefinitionType(a.DefinitionType),
		CategoryCode:   a.CategoryCode,
		TriggerType:    constants.AgentTriggerType(a.TriggerType),
		Name:           a.Name,
		Slug:           a.Slug,
		Description:    a.Description,
		Editability:    constants.EditabilityFromBool(a.IsEditable),
		AccountStatus:  accountStatus,
		CreatedAt:      timeutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:      timeutil.TimestampToTime(a.UpdatedAt),
	}
}

// StashAgentDefinitionMeta stashes the definition's expandable sub-resources (config, tools, and the
// role FK id) into the request-scoped load meta. The role is loaded with real data via LoadRoles only
// when ?include=role is requested; never fabricate role data here.
func StashAgentDefinitionMeta(meta *resourcekit.LoadMeta, a *pb.AgentDefinitionInfo) {
	if a == nil {
		return
	}

	meta.Set(constants.ObjectTypeAgentDefinition, a.Id, "config", unmarshalConfig(a.ConfigJson))

	toolItems := make([]apiresource.AgentDefinitionTool, len(a.Tools))
	for i, t := range a.Tools {
		toolItems[i] = apiresource.AgentDefinitionTool{
			ID:     t.Id,
			Object: constants.ObjectTypeAgentDefinitionTool,
			Tool: apiresource.AvailableTool{
				Slug:                t.ToolSlug,
				Object:              constants.ObjectTypeAvailableTool,
				Name:                t.DisplayName,
				Description:         &t.Description,
				ConfigSchema:        json.RawMessage(t.ConfigSchemaJson),
				Category:            t.Category,
				RequiredPermissions: orEmptyStrSlice(t.RequiredPermissions),
			},
			Config:            json.RawMessage(t.ConfigJson),
			SortOrder:         t.SortOrder,
			ReviewRequirement: constants.ReviewRequirementFromBool(t.RequireReview),
		}
	}
	meta.Set(constants.ObjectTypeAgentDefinition, a.Id, "tools", apiresource.NewList(toolItems, apiresource.PageInfo{}))

	// Role is an expandable sub-resource: stash only the FK id. When the client
	// requests ?include=role, the resourcekit resolver loads the real Role (and
	// its permissions/owner) via LoadRoles. Never fabricate role data here.
	if a.RoleId != "" {
		meta.Set(constants.ObjectTypeAgentDefinition, a.Id, "role_id", a.RoleId)
	}
}

func AgentDefinitionListPresenter(ctx context.Context, resp *pb.ListAgentDefinitionsResponse) *apiresource.List[apiresource.AgentDefinition] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentDefinition](nil, apiresource.PageInfo{})
	}

	meta := resourcekit.GetLoadMeta(ctx)
	agents := make([]apiresource.AgentDefinition, len(resp.Agents))
	for i, a := range resp.Agents {
		agents[i] = AgentDefinitionPresenter(a)
		StashAgentDefinitionMeta(meta, a)
	}

	return apiresource.NewList(agents, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func orEmptyStrSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
