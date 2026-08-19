package agents

import "github.com/augno/api/shared/constants"

func RegisterTools(registry *ToolHandlerRegistry) {
	registry.Register(string(constants.ToolCreateArtifact), HandleCreateArtifact)
	registry.Register(string(constants.ToolReadDoc), HandleReadDoc)
	registry.Register(string(constants.ToolFetchUrl), HandleFetchURL)
	registry.Register(string(constants.ToolSendEmail), HandleSendEmail)
	registry.Register(string(constants.ToolDraftReply), HandleDraftReply)
	registry.Register(FindAppPageSlug, HandleFindAppPage)

	// Generated tools that proxy to api-gateway endpoints flagged AgentTool=true.
	RegisterEndpointTools(registry)
}
