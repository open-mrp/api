package conversationep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// conversationIncludeFields is the whitelist of ?include= keys exposed by the conversation endpoints, resolved from what the service stashes into LoadMeta.
var conversationIncludeFields = []string{
	"assignee",
	"group",
	"participants",
	"related.record",
	"related.continues",
	"last_message",
	"last_message.sender",
	"last_message.author",
	"last_message.related.record",
	"last_message.attachments",
	"last_message.attachments.record",
}

func conversationIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeConversation,
		Fields:     conversationIncludeFields,
	})
}

// conversationLinkIncludeFields is the whitelist of ?include= keys exposed by the conversation-link endpoints. The linked conversation (and its own expandable sub-objects) is fetched by id via the chat batch loader.
var conversationLinkIncludeFields = []string{
	"record",
	"conversation",
	"conversation.assignee",
	"conversation.group",
	"conversation.participants",
	"conversation.related.record",
	"conversation.related.continues",
	"conversation.last_message",
	"conversation.last_message.sender",
	"conversation.last_message.author",
	"conversation.last_message.related.record",
	"conversation.last_message.attachments",
	"conversation.last_message.attachments.record",
}

func conversationLinkIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeConversationLink,
		Fields:     conversationLinkIncludeFields,
	})
}
