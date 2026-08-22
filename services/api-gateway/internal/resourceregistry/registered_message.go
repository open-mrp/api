package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeChatMessage,
		Load:       resourceloaders.LoadMessages,
		Subs: []resourcekit.SubField{
			// sender, author, resource and attachments arrive inline on the message payload and are stashed by chatmap.StashMessageMeta — gated visibility, no extra fetch.
			{Key: "sender", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateSenderOnMessage},
			{Key: "author", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateAuthorOnMessage},
			{Key: "resource", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateResourceOnMessage},
			// attachments arrive inline (stashed whole). Populate sets the list; Target+ExtractRefs let the resolver recurse into each attachment to gate its own expandable `resource`.
			{
				Key:         "attachments",
				Target:      constants.ObjectTypeMessageAttachment,
				Cardinality: resourcekit.CardinalityList,
				Populate:    populateAttachmentsOnMessage,
				ExtractRefs: extractAttachmentRefsFromMessage,
			},
			// conversation and reply_to are fetched by id via the chat batch loaders.
			{
				Key:         "conversation",
				Target:      constants.ObjectTypeConversation,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractConversationIDFromMessage,
				Populate:    populateConversationOnMessage,
			},
			{
				Key:         "reply_to",
				Target:      constants.ObjectTypeChatMessage,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractReplyToIDFromMessage,
				Populate:    populateReplyToOnMessage,
			},
			// agent_run is fetched by id via the agent-service run loader.
			{
				Key:         "agent_run",
				Target:      constants.ObjectTypeAgentRun,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractAgentRunIDFromMessage,
				Populate:    populateAgentRunOnMessage,
			},
		},
	})
}

func populateSenderOnMessage(ctx context.Context, parent any, _ map[string]any) {
	m := parent.(*apiresource.Message)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeChatMessage, m.ID, "sender")
	if !ok {
		return
	}
	m.Sender = v.(*apiresource.Actor)
}

func populateAuthorOnMessage(ctx context.Context, parent any, _ map[string]any) {
	m := parent.(*apiresource.Message)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeChatMessage, m.ID, "author")
	if !ok {
		return
	}
	m.Author = v.(*apiresource.Actor)
}

func populateResourceOnMessage(ctx context.Context, parent any, _ map[string]any) {
	m := parent.(*apiresource.Message)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeChatMessage, m.ID, "resource")
	if !ok {
		return
	}
	m.Resource = v.(*apiresource.Entity)
}

func populateAttachmentsOnMessage(ctx context.Context, parent any, _ map[string]any) {
	m := parent.(*apiresource.Message)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeChatMessage, m.ID, "attachments")
	if !ok {
		return
	}
	m.Attachments = v.(*apiresource.List[apiresource.MessageAttachment])
}

// extractAttachmentRefsFromMessage returns addressable pointers to the message's attachments so the resolver can recurse into them (for attachments.resource). Runs after populateAttachmentsOnMessage.
func extractAttachmentRefsFromMessage(_ context.Context, parent any) []any {
	m := parent.(*apiresource.Message)
	if m.Attachments == nil {
		return nil
	}
	refs := make([]any, 0, len(m.Attachments.Data))
	for i := range m.Attachments.Data {
		refs = append(refs, &m.Attachments.Data[i])
	}
	return refs
}

func extractConversationIDFromMessage(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Message)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeChatMessage, m.ID, "conversation_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateConversationOnMessage(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Message)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeChatMessage, m.ID, "conversation_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.Conversation = v.(*apiresource.Conversation)
	}
}

func extractReplyToIDFromMessage(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Message)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeChatMessage, m.ID, "reply_to_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateReplyToOnMessage(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Message)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeChatMessage, m.ID, "reply_to_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.ReplyTo = v.(*apiresource.Message)
	}
}

func extractAgentRunIDFromMessage(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Message)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeChatMessage, m.ID, "agent_run_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateAgentRunOnMessage(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Message)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeChatMessage, m.ID, "agent_run_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.AgentRun = v.(*apiresource.AgentRun)
	}
}
