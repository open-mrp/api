package agents

import "github.com/augno/api/shared/constants"

// This catalog declares the built-in agent tools entirely in code, mirroring the generated endpoint-tool catalog (EndpointTools). Built-in tools used to live as migration-seeded tool_definition/tool_group rows; they are now code-only, so adding, removing, or editing one is a pure code change with no database migration. The runtime handlers for these slugs are registered in register.go; an agent is granted a built-in tool by a slug entry in its agent_definition_tool links.

// BuiltinToolGroup is the display grouping for the tool-selection UI. These mirror the groups that used to be seeded into tool_group.
type BuiltinToolGroup struct {
	ID        string
	Name      string
	Slug      string
	Icon      string
	SortOrder int32
}

var (
	builtinGroupGeneral   = BuiltinToolGroup{ID: "tgrp_builtin_general", Name: "General", Slug: "general", Icon: "settings", SortOrder: 3}
	builtinGroupKnowledge = BuiltinToolGroup{ID: "tgrp_builtin_knowledge", Name: "Knowledge", Slug: "knowledge", Icon: "book", SortOrder: 4}
)

// BuiltinToolDescriptor describes one built-in agent tool: the runtime fields the LLM needs (Description, InputSchema) plus the metadata the tool-selection UI shows (DisplayName, Group, RequiredPermissions). Category is always "built_in".
type BuiltinToolDescriptor struct {
	Slug                constants.Tool
	DisplayName         string
	Description         string
	InputSchema         string
	Group               BuiltinToolGroup
	RequiredPermissions []string
	// Mutating reports whether invoking this tool takes an externally-visible or otherwise irreversible action (e.g. send_email puts mail in front of a customer). Mutating built-in tools default to requiring human review and that gate cannot be turned off per-agent — the run pauses in awaiting_approval whenever the agent calls one. Read-only tools and tools that only stage something for a human to approve (e.g. draft_reply) are non-mutating.
	Mutating bool
}

// BuiltinTools is the catalog of built-in agent tools. Regenerating is unnecessary — edit this slice directly. Each Slug must have a handler registered in RegisterTools (register.go).
var BuiltinTools = []BuiltinToolDescriptor{
	{
		Slug:        constants.ToolCreateArtifact,
		DisplayName: "Create Artifact",
		Description: "Create an artifact such as a report, document, or data export.",
		InputSchema: `{"type":"object","properties":{"artifact_type":{"type":"string","description":"Type of artifact (e.g., report, document, csv)"},"name":{"type":"string","description":"Artifact name"},"content":{"type":"string","description":"Artifact content"},"mime_type":{"type":"string","description":"MIME type of the content (e.g., text/plain, text/csv, application/json)"}},"required":["artifact_type","name","content","mime_type"]}`,
		Group:       builtinGroupGeneral,
	},
	{
		Slug:        constants.ToolReadDoc,
		DisplayName: "Read Doc",
		Description: "Read the content of an Augno documentation page. " +
			"To find the right page, first fetch https://docs.augno.com/llms.txt which lists all available pages with descriptions. " +
			"Then call this tool again with the URL of the page you want to read.",
		InputSchema: `{"type":"object","properties":{"url":{"type":"string","description":"The full URL of the documentation page to read (must be from docs.augno.com). Start with https://docs.augno.com/llms.txt to discover available pages."}},"required":["url"]}`,
		Group:       builtinGroupKnowledge,
	},
	{
		Slug:        constants.ToolFetchUrl,
		DisplayName: "Fetch URL",
		Description: "Fetch the content of a public URL. Returns the response body as text. Only HTTPS URLs are allowed. " +
			"When fetching a website for the first time, check if the site has an /llms.txt file (e.g. https://example.com/llms.txt). " +
			"This file follows the llms.txt standard and lists markdown-formatted URLs optimized for LLM consumption. " +
			"If llms.txt exists and contains relevant URLs, prefer fetching those markdown URLs instead of the raw HTML pages. " +
			"Also look for llms-full.txt for comprehensive content. Skip the llms.txt check for direct links to files, APIs, or non-website URLs.",
		InputSchema: `{"type":"object","properties":{"url":{"type":"string","description":"The HTTPS URL to fetch"}},"required":["url"]}`,
		Group:       builtinGroupKnowledge,
	},
	{
		Slug:        constants.ToolSendEmail,
		DisplayName: "Send Email",
		Description: "Send an email reply to the customer through the conversation's bound inbox. The reply goes to whoever last emailed this thread, threaded correctly. This is an externally-visible action. Only the subject, body, and optional cc are needed — the recipient is determined by the conversation.",
		InputSchema: `{"type":"object","properties":{"cc":{"type":"array","items":{"type":"string"},"description":"Optional cc addresses"},"subject":{"type":"string","description":"Email subject"},"body":{"type":"string","description":"Email body (plain text)"}},"required":["subject","body"]}`,
		Group:       builtinGroupGeneral,
		Mutating:    true,
	},
	{
		Slug:        constants.ToolDraftReply,
		DisplayName: "Draft Reply",
		Description: "Propose a reply to the external party on a case — whoever the case corresponds with (a customer, supplier, or other contact reachable over the case's inbox). The draft is held for a human teammate to review, edit, and approve before it is sent — it is NOT sent by this tool. Use it for BOTH channels: if the case is bridged to an email inbox the approved draft is sent as an email reply (set the optional subject); otherwise it is sent as an in-app portal message. The channel is chosen automatically from the case, so you do not need a separate email tool. Provide just the reply body (and, for an email case, an optional subject); you do not need — and will not be given — a conversation ID. Use this to answer the person on the other end; write the message exactly as it should be sent to them.",
		InputSchema: `{"type":"object","properties":{"body":{"type":"string","description":"The outbound reply text, written exactly as it should be sent to the recipient."},"subject":{"type":"string","description":"Optional email subject; used only when the case is email-bridged, ignored otherwise."}},"required":["body"]}`,
		Group:       builtinGroupGeneral,
	},
}

var builtinToolIndex = func() map[constants.Tool]BuiltinToolDescriptor {
	m := make(map[constants.Tool]BuiltinToolDescriptor, len(BuiltinTools))
	for _, d := range BuiltinTools {
		m[d.Slug] = d
	}
	return m
}()

// LookupBuiltinTool returns the catalog descriptor for a slug.
func LookupBuiltinTool(slug string) (BuiltinToolDescriptor, bool) {
	d, ok := builtinToolIndex[constants.Tool(slug)]
	return d, ok
}

// BuiltinToolGroups returns the distinct display groups referenced by BuiltinTools, in catalog order.
func BuiltinToolGroups() []BuiltinToolGroup {
	seen := map[string]bool{}
	var out []BuiltinToolGroup
	for _, d := range BuiltinTools {
		if d.Group.ID == "" || seen[d.Group.ID] {
			continue
		}
		seen[d.Group.ID] = true
		out = append(out, d.Group)
	}
	return out
}
