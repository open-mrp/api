package apiresource

import (
	"encoding/json"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleAgentDefinitionToolID = "agdftl_0108b30efc261d3d47bf1b43a1"
const SampleAvailableToolSlug = constants.ToolReadDoc

// Logical grouping of platform tools.
type ToolGroup struct {
	// Group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tool_group"`
	// Human-readable group name (e.g. `Product Tools`).
	Name string `json:"name" validate:"required"`
	// Description of what the tools in this group do.
	Description *string `json:"description"`
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
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=available_tool"`
	// A stable identifier used when attaching the tool to an agent.
	Slug string `json:"slug" validate:"required"`
	// Category grouping for the tool (e.g. `built_in`).
	Category string `json:"category" validate:"required"`
	// Tool name.
	Name string `json:"name" validate:"required"`
	// Tool description.
	Description *string `json:"description"`
	// JSON schema describing the configuration options this tool accepts.
	//
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
	// Permission scopes the agent's role must hold for this tool to be usable (e.g. `products:read`).
	RequiredPermissions []string `json:"required_permissions"`
	// Role type the caller must have for this tool, when the operation is gated by role rather than a permission (e.g. `admin`).
	RequiredRoleType *string `json:"required_role_type"`
	// Whether invoking this tool changes server state.
	//
	// True for any `api_endpoint` tool whose underlying operation is not a read (non-GET); always false for `built_in` tools. The agent-configuration UI uses this to default such tools to requiring human review.
	Mutating bool `json:"mutating"`
}

// Tool attached to an agent definition.
//
// Pairs an AvailableTool with agent-specific config values.
type AgentDefinitionTool struct {
	// Agent definition tool ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_definition_tool"`
	// Attached tool.
	Tool AvailableTool `json:"tool" validate:"required"`
	// Whether calls to this tool must be approved by a user before they execute.
	//
	// When `required`, the run pauses in the `awaiting_approval` status each time the agent invokes this tool; approve or allow the tool via the Continue Agent Run endpoint to proceed.
	ReviewRequirement constants.ReviewRequirement `json:"review_requirement" validate:"required"`
	// Instance-specific configuration for this tool.
	//
	// Must conform to the tool's `config_schema`.
	Config json.RawMessage `json:"config"`
	// Sort order within the agent.
	SortOrder int32 `json:"sort_order"`
}

const SampleToolGroupID = "tgrp_01556abdc283b09ccd1f97dcb5"

var SampleToolGroup = &ToolGroup{
	ID:          SampleToolGroupID,
	Object:      constants.ObjectTypeToolGroup,
	Name:        "Customer Tools",
	Description: new("Tools for looking up and managing customers."),
	Slug:        "customer_tools",
	Icon:        "people",
	SortOrder:   0,
	Tools:       NewList([]AvailableTool{*SampleAvailableTool}, PageInfo{}),
}

var SampleAvailableTool = &AvailableTool{
	Object:              constants.ObjectTypeAvailableTool,
	Slug:                string(SampleAvailableToolSlug),
	Category:            "built_in",
	Name:                "Lookup Customer",
	Description:         new("Look up a customer by their email address."),
	ConfigSchema:        nil,
	RequiredPermissions: []string{"customers:read"},
}

var SampleAgentDefinitionTool = &AgentDefinitionTool{
	ID:                SampleAgentDefinitionToolID,
	Object:            constants.ObjectTypeAgentDefinitionTool,
	Tool:              *SampleAvailableTool,
	ReviewRequirement: constants.ReviewRequirementNotRequired,
	Config:            json.RawMessage(`{}`),
	SortOrder:         0,
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
