package conversationep

import (
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
)

// conversationIncludeFields is the whitelist of ?include= keys exposed by the conversation endpoints.
// assignee, group, participants, topic and last_message (with its own sub-objects) are
// expandable — gated here and resolved by the resourcekit registry from the data the service stashes
// into LoadMeta.
var conversationIncludeFields = []string{
	"assignee",
	"group",
	"participants",
	"topic",
	"last_message",
	"last_message.sender",
	"last_message.author",
	"last_message.resource",
	"last_message.attachments",
	"last_message.attachments.resource",
}

func conversationIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeConversation,
		Fields:     conversationIncludeFields,
	})
}

// conversationLinkIncludeFields is the whitelist of ?include= keys exposed by the conversation-link
// endpoints. The linked conversation (and its own expandable sub-objects) is fetched by id via the
// chat batch loader.
var conversationLinkIncludeFields = []string{
	"conversation",
	"conversation.assignee",
	"conversation.group",
	"conversation.participants",
	"conversation.topic",
	"conversation.last_message",
	"conversation.last_message.sender",
	"conversation.last_message.author",
	"conversation.last_message.resource",
	"conversation.last_message.attachments",
	"conversation.last_message.attachments.resource",
}

func conversationLinkIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeConversationLink,
		Fields:     conversationLinkIncludeFields,
	})
}
