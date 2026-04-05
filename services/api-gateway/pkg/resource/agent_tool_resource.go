package apiresource

import (
	"encoding/json"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// ToolGroup represents a logical grouping of platform tools.
type ToolGroup struct {
	// The unique identifier for the group.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tool_group"`
	// The display name of the group.
	Name string `json:"name" validate:"required"`
	// A description of the tool group.
	Description string `json:"description"`
	// A URL-friendly slug for the group.
	Slug string `json:"slug" validate:"required"`
	// An icon identifier for the group (e.g. a Material Icon name).
	Icon string `json:"icon"`
	// Sort order for display purposes.
	SortOrder int32 `json:"sort_order"`
	// The tools belonging to this group.
	Tools *List[AvailableTool] `json:"tools" expandable:"true"`
}

// AvailableTool represents a platform tool that can be attached to agents.
// Each tool has a config_schema that describes what configuration options it accepts,
// and an input_schema (internal, not exposed via API) that tells the LLM what arguments
// to pass when invoking the tool at runtime.
type AvailableTool struct {
	// The unique identifier for the tool.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=available_tool"`
	// The name of the tool.
	Name string `json:"name" validate:"required"`
	// A description of the tool.
	Description *string `json:"description"`
	// A JSON schema describing what configuration options this tool accepts.
	// Defines the shape of the `config` field on AgentDefinitionTool.
	//
	// For example:
	//
	// ```json
	// {
	//   "type": "object",
	//   "properties": {
	//     "max_results": {
	//       "type": "integer",
	//       "default": 10
	//     }
	//   }
	// }
	// ```
	ConfigSchema json.RawMessage `json:"config_schema"`
	// The tool category.
	Category string `json:"category" validate:"required"`
	// Permissions required to use this tool.
	RequiredPermissions []string `json:"required_permissions"`
}

// AgentDefinitionTool represents a tool attached to an agent definition.
// It pairs an AvailableTool with agent-specific configuration and settings.
// The tool's config_schema defines what options are available; the config field
// here holds the actual values chosen for this particular agent.
// Different agents using the same tool can have different config values.
type AgentDefinitionTool struct {
	// The unique identifier for this agent-tool link.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition_tool"`
	// The tool attached to this agent definition.
	Tool AvailableTool `json:"tool" validate:"required"`
	// The instance-specific configuration values for this tool on this agent.
	// Must conform to the tool's config_schema. These values are used by the
	// tool handler at runtime but are not exposed to the LLM.
	Config json.RawMessage `json:"config"`
	// The sort order of this tool within the agent.
	SortOrder int32 `json:"sort_order"`
	// Whether this tool requires human review before execution.
	RequireReview bool `json:"require_review"`
}

var SampleToolGroup = &ToolGroup{
	ID:          "tgrp_01k0b1seed0product000000",
	Object:      constants.ObjectTypeToolGroup,
	Name:        "Product Tools",
	Description: "Tools for searching and managing products.",
	Slug:        "product_tools",
	Icon:        "inventory",
	SortOrder:   1,
}

var SampleAvailableTool = &AvailableTool{
	ID:                  SampleAvailableToolID,
	Object:              constants.ObjectTypeAvailableTool,
	Name:                "Search Products",
	Description:         new("Search for products by keyword or phrase"),
	ConfigSchema:        nil,
	Category:            "built_in",
	RequiredPermissions: []string{"products:read"},
}

var SampleAgentDefinitionTool = &AgentDefinitionTool{
	ID:            SampleAgentDefinitionToolID,
	Object:        constants.ObjectTypeAgentDefinitionTool,
	Tool:          *SampleAvailableTool,
	Config:        json.RawMessage(`{}`),
	SortOrder:     0,
	RequireReview: false,
}

func (*AgentDefinitionTool) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentDefinitionTool)
}

func (*AvailableTool) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAvailableTool)
}

func (*ToolGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleToolGroup)
}
