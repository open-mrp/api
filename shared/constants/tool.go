package constants

// Tool represents the slug identifier for a built-in agent tool.
type Tool string

const (
	// ToolCreateArtifact creates an artifact (report, document, data export).
	ToolCreateArtifact Tool = "create_artifact"
	// ToolReadDoc reads OpenMRP documentation pages.
	ToolReadDoc Tool = "read_doc"
	// ToolSearchDocs searches the OpenMRP documentation index for pages matching a query.
	ToolSearchDocs Tool = "search_docs"
	// ToolDescribeApiOperation returns the input schema, includes, permissions, and route of an API operation.
	ToolDescribeApiOperation Tool = "describe_api_operation"
	// ToolSearchSource searches the platform's public source code at the deployed version.
	ToolSearchSource Tool = "search_source"
	// ToolReadSource reads a file of the platform's public source code at the deployed version.
	ToolReadSource Tool = "read_source"
	// ToolFetchUrl fetches content from a public URL.
	ToolFetchUrl Tool = "fetch_url"
	// ToolSendEmail sends an email reply through the conversation's bound inbox (gated by human review).
	ToolSendEmail Tool = "send_email"
	// ToolDraftReply proposes a reply to the external party on a case (a customer, supplier, or any
	// contact the case corresponds with) as a draft held for human approval — channel resolved from the
	// case: email if bridged to an inbox, else a portal message. Does not send.
	ToolDraftReply Tool = "draft_reply"
)

func (s Tool) IsValid() bool {
	switch s {
	case ToolCreateArtifact, ToolReadDoc, ToolSearchDocs, ToolDescribeApiOperation, ToolSearchSource, ToolReadSource,
		ToolFetchUrl, ToolSendEmail, ToolDraftReply:
		return true
	default:
		return false
	}
}

func (s Tool) EnumValues() []string {
	return []string{
		string(ToolCreateArtifact),
		string(ToolReadDoc),
		string(ToolSearchDocs),
		string(ToolDescribeApiOperation),
		string(ToolSearchSource),
		string(ToolReadSource),
		string(ToolFetchUrl),
		string(ToolSendEmail),
		string(ToolDraftReply),
	}
}
