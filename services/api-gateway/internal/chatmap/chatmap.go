// Package chatmap converts notification-service chat proto messages into gateway API resources.
// It is shared by the messaging endpoint packages (conversations, messages, participants, scheduled messages, attachments, blocks) and the resource loaders so the proto→resource mapping lives in one place.
//
// The base mappers (ConversationFromProto, MessageFromProto) populate only inline fields and leave every expandable sub-object (assignee, group, topic, participants, last_message, sender, author, attachments, resource, reply_to, conversation) nil. The Stash* helpers build those sub-objects and stash them in the request-scoped LoadMeta so the include resolver can attach them only when the caller requests them via ?include=. Name hydration is layered on top in the endpoint services / loaders (chatmap must not import resourceloaders — that would be an import cycle).
package chatmap

import (
	"context"
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/notification"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConversationStatusFromProto maps the proto status + per-caller hidden flag to the resource enum.
// Hidden takes precedence over the account-level status; an empty status defaults to active.
func ConversationStatusFromProto(status string, hidden bool) constants.ConversationStatus {
	if hidden {
		return constants.ConversationStatusHidden
	}
	if status == "" {
		return constants.ConversationStatusActive
	}
	return constants.ConversationStatus(status)
}

// OkMessage builds a human-readable confirmation for actions that produce no resource.
func OkMessage(msg string) *apiresource.MessageResource {
	return &apiresource.MessageResource{Object: constants.ObjectTypeMessage, Message: msg}
}

// ConversationFromProto maps a conversation to its base API resource. Expandable sub-objects (assignee, group, topic, participants, last_message) are left nil — call StashConversationMeta to make them resolvable via ?include=.
func ConversationFromProto(c *pb.ConversationInfo) apiresource.Conversation {
	if c == nil {
		return apiresource.Conversation{}
	}
	conv := apiresource.Conversation{
		ID:            c.Id,
		Object:        constants.ObjectTypeConversation,
		Type:          constants.ConversationType(c.Type),
		Audience:      constants.ConversationAudience(c.Audience),
		Title:         c.Title,
		Status:        ConversationStatusFromProto(c.Status, c.Hidden),
		LegalHold:     constants.LegalHoldStatusFromHeld(c.LegalHold),
		Unread:        c.Unread,
		LastMessageAt: TsToPtr(c.LastMessageAt),
		CreatedAt:     TsToTime(c.CreatedAt),
		UpdatedAt:     TsToTime(c.UpdatedAt),
	}
	if c.WorkflowStatus != nil && *c.WorkflowStatus != "" {
		ws := constants.ConversationWorkflowStatus(*c.WorkflowStatus)
		conv.WorkflowStatus = &ws
	}
	return conv
}

// StashConversationMeta builds the conversation's expandable sub-objects (assignee, group, topic, participants, last_message) and stashes them in the request LoadMeta so the include resolver can attach them.
func StashConversationMeta(ctx context.Context, c *pb.ConversationInfo, d *apiresource.Conversation) {
	if c == nil || d == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	ot := constants.ObjectTypeConversation

	// The assignee is a single polymorphic case owner — a user or a team. Map the stored resource type
	// (account_user / account_group) onto the matching actor type so the client sees one assignee field.
	if c.AssigneeResourceId != nil && *c.AssigneeResourceId != "" {
		assigneeType := constants.ActorTypeUser
		if c.AssigneeResourceType != nil && constants.ObjectType(*c.AssigneeResourceType) == constants.ObjectTypeAccountGroup {
			assigneeType = constants.ActorTypeGroup
		}
		meta.Set(ot, d.ID, "assignee", apiresource.NewActor(*c.AssigneeResourceId, assigneeType, nil, nil))
	}
	if c.GroupId != nil && *c.GroupId != "" {
		meta.Set(ot, d.ID, "group", &apiresource.MessagingGroup{ID: *c.GroupId, Object: constants.ObjectTypeMessagingGroup})
	}

	participants := make([]apiresource.ConversationParticipant, 0, len(c.Participants))
	for _, p := range c.Participants {
		participants = append(participants, ParticipantFromProto(p))
	}
	meta.Set(ot, d.ID, "participants", apiresource.NewList(participants, apiresource.PageInfo{}))

	if c.TopicResourceType != nil && c.TopicResourceId != nil {
		meta.Set(ot, d.ID, "topic", apiresource.NewEntity(*c.TopicResourceId, constants.ObjectType(*c.TopicResourceType), nil, nil))
	}

	if c.LastMessage != nil {
		lm := MessageFromProto(c.LastMessage)
		StashMessageMeta(ctx, c.LastMessage, &lm)
		meta.Set(ot, d.ID, "last_message", &lm)
	}
}

// ParticipantFromProto maps a conversation participant to its API resource. The actor carries id + type only; display names are hydrated downstream.
func ParticipantFromProto(p *pb.ParticipantInfo) apiresource.ConversationParticipant {
	result := apiresource.ConversationParticipant{
		ID:            p.Id,
		Object:        constants.ObjectTypeConversationParticipant,
		Type:          constants.ParticipantType(p.ParticipantType),
		Role:          constants.ParticipantRole(p.Role),
		Membership:    constants.ParticipantMembership(p.Membership),
		Notifications: constants.ParticipantNotifications(p.Notifications),
		ReadCursor: apiresource.ReadCursor{
			Object:    constants.ObjectTypeReadCursor,
			Sequence:  p.LastReadSequence,
			MessageID: p.LastReadMessageId,
		},
	}
	if p.LastReadAt != nil {
		t := p.LastReadAt.AsTime()
		result.ReadCursor.ReadAt = &t
	}
	// One polymorphic actor: a user or an agent. A customer participant (an external contact on a
	// support case) carries the account_user of the person who opened the case, so it resolves through
	// the user branch — its relation_account_id is a routing/dedup key only, never a surfaced actor.
	// System participants have no actor.
	if p.AccountUserId != nil && *p.AccountUserId != "" {
		var name *string
		if p.AccountUserDisplayName != nil && *p.AccountUserDisplayName != "" {
			name = p.AccountUserDisplayName
		}
		result.Actor = apiresource.NewActor(*p.AccountUserId, constants.ActorTypeUser, name, nil)
	} else if p.AgentConfigId != nil && *p.AgentConfigId != "" {
		result.Actor = apiresource.NewActor(*p.AgentConfigId, constants.ActorTypeAgent, nil, nil)
	}
	if p.AgentTriggerPolicy != nil && *p.AgentTriggerPolicy != "" {
		policy := constants.AgentTriggerPolicy(*p.AgentTriggerPolicy)
		result.AgentTriggerPolicy = &policy
	}
	result.AgentTriggerKeywords = p.AgentTriggerKeywords
	return result
}

// ConversationLinkFromProto maps a conversation link to its base API resource (the linked record as a
// bare id+type Entity; the display name is hydrated downstream where needed). The expandable
// `conversation` sub-object is left nil — call StashConversationLinkMeta to make it resolvable via
// ?include=conversation.
func ConversationLinkFromProto(l *pb.ConversationLinkInfo) apiresource.ConversationLink {
	if l == nil {
		return apiresource.ConversationLink{}
	}
	return apiresource.ConversationLink{
		ID:        l.Id,
		Object:    constants.ObjectTypeConversationLink,
		Resource:  apiresource.NewEntity(l.ResourceId, constants.ObjectType(l.ResourceType), nil, nil),
		CreatedAt: TsToTime(l.CreatedAt),
	}
}

// StashConversationLinkMeta stashes the link's conversation FK so ?include=conversation fetches the
// full conversation by id via the chat batch loader (which in turn stashes the conversation's own
// sub-objects, letting conversation.* recurse).
func StashConversationLinkMeta(ctx context.Context, l *pb.ConversationLinkInfo, d *apiresource.ConversationLink) {
	if l == nil || d == nil {
		return
	}
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeConversationLink, d.ID, "conversation_id", l.ConversationId)
}

// MessageFromProto maps a chat message to its base API resource. Expandable sub-objects (conversation, sender, author, resource, reply_to, attachments) are left nil — call StashMessageMeta to make them resolvable via ?include=.
func MessageFromProto(m *pb.MessageInfo) apiresource.Message {
	if m == nil {
		return apiresource.Message{}
	}
	msg := apiresource.Message{
		ID:              m.Id,
		Object:          constants.ObjectTypeChatMessage,
		Kind:            constants.MessageKind(m.Kind),
		Status:          constants.MessageStatus(m.Status),
		Visibility:      constants.MessageVisibility(m.Visibility),
		Body:            m.Body,
		Subject:         m.Subject,
		ScheduledAt:     TsToPtr(m.ScheduledFor),
		StreamingState:  m.StreamingState,
		ClientMessageID: m.ClientMessageId,
		EditedAt:        TsToPtr(m.EditedAt),
		DeletedAt:       TsToPtr(m.DeletedAt),
		CreatedAt:       TsToTime(m.CreatedAt),
		UpdatedAt:       TsToTime(m.UpdatedAt),
		Channel:         constants.ResolveMessageChannel(m.Channel, m.Kind),
	}
	if m.Status == "" {
		msg.Status = constants.MessageStatusSent
	}
	msg.Sequence = m.Sequence
	return msg
}

// StashMessageMeta builds the message's expandable sub-objects (sender, author, resource, attachments) plus the conversation/reply_to foreign keys and stashes them in the request LoadMeta for the include resolver. Sender/author actors carry id + type only; names are hydrated downstream.
func StashMessageMeta(ctx context.Context, m *pb.MessageInfo, d *apiresource.Message) {
	if m == nil || d == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	ot := constants.ObjectTypeChatMessage

	attachments := make([]apiresource.MessageAttachment, 0, len(m.Attachments))
	for _, a := range m.Attachments {
		attachments = append(attachments, AttachmentFromProto(ctx, a))
	}
	meta.Set(ot, d.ID, "attachments", apiresource.NewList(attachments, apiresource.PageInfo{}))

	if sender := SenderActorFromProto(m); sender != nil {
		meta.Set(ot, d.ID, "sender", sender)
	}
	if author := AuthorActorFromProto(m); author != nil {
		meta.Set(ot, d.ID, "author", author)
	}
	if m.LinkResourceType != nil && m.LinkResourceId != nil {
		meta.Set(ot, d.ID, "resource", apiresource.NewEntity(*m.LinkResourceId, constants.ObjectType(*m.LinkResourceType), nil, nil))
	}
	meta.Set(ot, d.ID, "conversation_id", m.ConversationId)
	if m.ReplyToMessageId != nil && *m.ReplyToMessageId != "" {
		meta.Set(ot, d.ID, "reply_to_id", *m.ReplyToMessageId)
	}
	// agent_run is expandable: stash the FK so ?include=agent_run fetches the run by id.
	if m.AgentRunId != nil && *m.AgentRunId != "" {
		meta.Set(ot, d.ID, "agent_run_id", *m.AgentRunId)
	}
}

// AuthorActorFromProto builds the underlying author actor — an account user or an agent. It is nil for system messages and when an anonymizing sender identity hides the real author (the server strips the backing ids in that case).
func AuthorActorFromProto(m *pb.MessageInfo) *apiresource.Actor {
	if m.SenderAccountUserId != nil && *m.SenderAccountUserId != "" {
		var name *string
		if m.SenderDisplayName != nil && *m.SenderDisplayName != "" {
			name = m.SenderDisplayName
		}
		return apiresource.NewActor(*m.SenderAccountUserId, constants.ActorTypeUser, name, nil)
	}
	if m.SenderAgentConfigId != nil && *m.SenderAgentConfigId != "" {
		return apiresource.NewActor(*m.SenderAgentConfigId, constants.ActorTypeAgent, nil, nil)
	}
	return nil
}

// customerServiceActorID is the stable synthetic id of the branded "Customer Service" party a customer
// sees behind staff replies on an external case. The real author is anonymized upstream (the notification
// service clears sender_account_user_id / sender_agent_config_id for a customer-relation viewer).
const customerServiceActorID = "customer_service"

// SenderActorFromProto builds the displayed sender actor: the branded "Customer Service" alias when the
// message was collapsed for a customer viewer, otherwise the authoring user or agent. Nil for pure system messages.
func SenderActorFromProto(m *pb.MessageInfo) *apiresource.Actor {
	if m.SenderAliasName != nil && *m.SenderAliasName != "" {
		return apiresource.NewActor(customerServiceActorID, constants.ActorTypeGroup, m.SenderAliasName, nil)
	}
	if m.SenderAccountUserId != nil && *m.SenderAccountUserId != "" {
		var name *string
		if m.SenderDisplayName != nil && *m.SenderDisplayName != "" {
			name = m.SenderDisplayName
		}
		return apiresource.NewActor(*m.SenderAccountUserId, constants.ActorTypeUser, name, nil)
	}
	if m.SenderAgentConfigId != nil && *m.SenderAgentConfigId != "" {
		return apiresource.NewActor(*m.SenderAgentConfigId, constants.ActorTypeAgent, nil, nil)
	}
	// An inbound email's sender is an external customer with no account_user/agent id — only a resolved
	// display name ("Name <addr>") the notification service lifted from the mail headers. Surface it as a
	// synthetic party so the timeline shows who the email is from rather than an unattributed bubble.
	if m.SenderDisplayName != nil && *m.SenderDisplayName != "" {
		return apiresource.NewActor(externalEmailSenderActorID, constants.ActorTypeGroup, m.SenderDisplayName, nil)
	}
	return nil
}

// externalEmailSenderActorID is the stable synthetic id for the sender of an inbound email — an external
// party with no account_user id. Like customerServiceActorID it is a display-only actor (non-navigable).
const externalEmailSenderActorID = "external_email_sender"

// AttachmentFromProto maps a message attachment to its API resource.
func AttachmentFromProto(ctx context.Context, a *pb.AttachmentInfo) apiresource.MessageAttachment {
	result := apiresource.MessageAttachment{
		ID:          a.Id,
		Object:      constants.ObjectTypeMessageAttachment,
		Kind:        constants.MessageAttachmentKind(a.Kind),
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		URL:         a.Url,
		CreatedAt:   TsToTime(a.CreatedAt),
	}
	if a.ResourceType != nil && *a.ResourceType != "" && a.ResourceId != nil && *a.ResourceId != "" {
		// resource is expandable: left nil on the base attachment, stashed for ?include=attachments.resource.
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeMessageAttachment, result.ID, "resource",
			apiresource.NewEntity(*a.ResourceId, constants.ObjectType(*a.ResourceType), nil, nil))
	}
	return result
}

// TsToTime converts a proto timestamp to time.Time (zero value when nil).
func TsToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// TsToPtr converts a proto timestamp to *time.Time (nil when absent).
func TsToPtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
