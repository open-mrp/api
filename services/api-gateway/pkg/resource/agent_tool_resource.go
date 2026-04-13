package apiresource

import (
	"encoding/json"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Logical grouping of platform tools.
type ToolGroup struct {
	// Group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tool_group"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Description.
	Description string `json:"description"`
	// URL-friendly slug.
	Slug string `json:"slug" validate:"required"`
	// Icon identifier (e.g. a Material Icon name).
	Icon string `json:"icon"`
	// Display sort order.
	SortOrder int32 `json:"sort_order"`
	// Tools belonging to this group.
	Tools *List[AvailableTool] `json:"tools" expandable:"true"`
}

// Platform tool that can be attached to agents.
type AvailableTool struct {
	// Tool ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=available_tool"`
	// Tool name.
	Name string `json:"name" validate:"required"`
	// Tool description.
	Description *string `json:"description"`
	// JSON schema describing the configuration options this tool accepts.
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
	// Tool category.
	Category string `json:"category" validate:"required"`
	// Required permissions.
	RequiredPermissions []string `json:"required_permissions"`
}

// Tool attached to an agent definition.
// Pairs an AvailableTool with agent-specific config values.
type AgentDefinitionTool struct {
	// Agent definition tool ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition_tool"`
	// Attached tool.
	Tool AvailableTool `json:"tool" validate:"required"`
	// Instance-specific configuration for this tool.
	// Must conform to the tool's config_schema.
	Config json.RawMessage `json:"config"`
	// Sort order within the agent.
	SortOrder int32 `json:"sort_order"`
	// Requires human review before execution.
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
