package apiresource

import (
	"encoding/json"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

const SampleAgentDefinitionToolID = "agdftl_iyc1asmsg1pu"
const SampleAvailableToolSlug = constants.ToolReadDoc

// A named grouping of the tools that can be granted to an agent, used to organize the tool catalog.
type ToolGroup struct {
	// Group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tool_group"`
	// Human-readable group name (e.g. `Product Tools`).
	Name string `json:"name" validate:"required"`
	// Description of what the tools in this group do.
	Description *string `json:"description"`
	// Machine-readable name for the group (e.g. `customer_tools`).
	Slug string `json:"slug" validate:"required"`
	// Icon identifier (e.g. a Material Icon name).
	Icon string `json:"icon"`
	// Display sort order, lowest first.
	SortOrder int32 `json:"sort_order"`
	// Tools belonging to this group.
	Tools *List[AvailableTool] `json:"tools" expandable:"true"`
}

// A capability an agent can be granted, allowing it to take that action during a run.
//
// The catalog of available tools is the same for every account; granting one to an agent is what makes it callable.
type AvailableTool struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=available_tool"`
	// A stable identifier used when attaching the tool to an agent.
	Slug string `json:"slug" validate:"required"`
	// Where the tool's behavior comes from.
	//
	// - `built_in`: a capability implemented by the agent runtime itself, such as fetching a web page or drafting a reply for a teammate to approve.
	// - `api_endpoint`: an operation of this API exposed as a tool, letting the agent perform it on the account's behalf.
	Category string `json:"category" validate:"required"`
	// Human-readable name for the tool.
	Name string `json:"name" validate:"required"`
	// Explanation of what the tool does.
	//
	// This is also the description the agent's model reads when deciding whether to call the tool.
	Description *string `json:"description"`
	// JSON schema describing the configuration options this tool accepts.
	//
	// Defines the shape of the `config` field on AgentDefinitionTool: a schema declaring a `max_results` integer property means that tool's `config` may set `max_results`.
	ConfigSchema json.RawMessage `json:"config_schema"`
	// Permission scopes the agent's role must hold for this tool to be usable (e.g. `products:read`).
	RequiredPermissions []string `json:"required_permissions"`
	// Role type the caller must have for this tool, when the operation is gated by role rather than a permission (e.g. `admin`).
	RequiredRoleType *string `json:"required_role_type"`
	// Whether invoking this tool takes an action rather than only reading data.
	//
	// True for any `api_endpoint` tool whose underlying operation is not a read, and for `built_in` tools that do something externally visible or hard to undo, such as sending an email. A mutating `built_in` tool always pauses its run for human approval and that gate cannot be turned off for an individual agent; for `api_endpoint` tools the flag is advisory and review stays configurable per agent.
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
	// When `required`, the run pauses in the `awaiting_approval` status each time the agent invokes this tool; approve or allow the tool via the Continue Agent Run endpoint to proceed. A tool whose `mutating` flag is true still pauses for approval even when this is `not_required`.
	ReviewRequirement constants.ReviewRequirement `json:"review_requirement" validate:"required"`
	// Instance-specific configuration for this tool.
	//
	// Must conform to the tool's `config_schema`.
	Config json.RawMessage `json:"config"`
	// Sort order within the agent.
	SortOrder int32 `json:"sort_order"`
}

const SampleToolGroupID = "tgrp_imjaqprzuqv5"

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
