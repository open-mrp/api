package messageep

import (
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
)

// messageIncludeFields is the whitelist of ?include= keys exposed by the message endpoints. sender,
// author, resource and attachments are stashed inline by the service; conversation and reply_to are
// fetched by id via the chat batch loaders.
var messageIncludeFields = []string{
	"sender",
	"author",
	"resource",
	"attachments",
	"attachments.resource",
	"conversation",
	"conversation.participants",
	"conversation.last_message",
	"reply_to",
	"reply_to.sender",
	"reply_to.author",
	"reply_to.attachments",
	"agent_run",
}

func messageIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeChatMessage,
		Fields:     messageIncludeFields,
	})
}
