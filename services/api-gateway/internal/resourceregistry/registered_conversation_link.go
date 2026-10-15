package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeConversationLink,
		Load:       resourceloaders.LoadConversationLinks,
		Subs: []include.SubField{
			messagingRecordSub("record", constants.ObjectTypeConversationLink,
				func(p any) string { return p.(*apiresource.ConversationLink).ID }, chatmap.MetaRecord,
				func(p any, r *apiresource.Record) { p.(*apiresource.ConversationLink).Record = r }),
			// conversation is fetched by id via the chat batch loader; its own expandable sub-objects (assignee, participants, last_message, …) recurse from the conversation registration.
			{
				Key:         "conversation",
				Target:      constants.ObjectTypeConversation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractConversationIDFromLink,
				Populate:    populateConversationOnLink,
			},
		},
	})
}

func extractConversationIDFromLink(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.ConversationLink)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeConversationLink, l.ID, "conversation_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateConversationOnLink(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.ConversationLink)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeConversationLink, l.ID, "conversation_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Conversation = v.(*apiresource.Conversation)
	}
}
