package chatmap

import (
	"context"
	"testing"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/notification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func strptr(s string) *string { return &s }

func metaCtx() context.Context { return resourcekit.WithLoadMeta(context.Background()) }

func TestConversationStatusFromProto(t *testing.T) {
	assert.Equal(t, constants.ConversationStatusActive, ConversationStatusFromProto("", false), "empty defaults to active")
	assert.Equal(t, constants.ConversationStatusActive, ConversationStatusFromProto("active", false))
	assert.Equal(t, constants.ConversationStatusArchived, ConversationStatusFromProto("archived", false))
	assert.Equal(t, constants.ConversationStatusHidden, ConversationStatusFromProto("active", true), "hidden takes precedence")
	assert.Equal(t, constants.ConversationStatusHidden, ConversationStatusFromProto("archived", true), "hidden overrides archived")
}

func TestConversationFromProto_BaseLeavesExpandablesNil(t *testing.T) {
	now := timestamppb.Now()
	info := &pb.ConversationInfo{
		Id: "cv_1", AccountId: "ac_1", Type: "group", Title: strptr("Team"),
		TopicResourceType: strptr("sales_order"), TopicResourceId: strptr("so_9"),
		AssigneeResourceType: strptr(string(constants.ObjectTypeAccountUser)), AssigneeResourceId: strptr("acus_9"), GroupId: strptr("mgrp_9"),
		Status: "archived", Unread: 4, LastMessageAt: now, CreatedAt: now, UpdatedAt: now,
		Participants: []*pb.ParticipantInfo{
			{Id: "cvpt_1", ConversationId: "cv_1", AccountId: "ac_1", ParticipantType: "user",
				AccountUserId: strptr("acus_1"), Role: "owner", Membership: "active", Notifications: "muted"},
		},
	}
	c := ConversationFromProto(info)
	assert.Equal(t, "cv_1", c.ID)
	assert.Equal(t, constants.ObjectTypeConversation, c.Object)
	assert.Equal(t, constants.ConversationTypeGroup, c.Type)
	assert.Equal(t, constants.ConversationStatusArchived, c.Status)
	assert.Equal(t, int64(4), c.Unread)
	require.NotNil(t, c.LastMessageAt, "last_message_at is inline, not expandable")
	assert.Nil(t, c.Assignee, "assignee is expandable — nil until stashed+resolved")
	assert.Nil(t, c.Group, "group is expandable — nil until stashed+resolved")
	assert.Nil(t, c.Topic, "topic is expandable — nil until stashed+resolved")
	assert.Nil(t, c.Participants, "participants are expandable — nil until stashed+resolved")
	assert.Nil(t, c.LastMessage, "last_message is expandable — nil until stashed+resolved")
}

func TestStashConversationMeta(t *testing.T) {
	ctx := metaCtx()
	now := timestamppb.Now()
	info := &pb.ConversationInfo{
		Id: "cv_1", AccountId: "ac_1", Type: "group", Status: "active",
		TopicResourceType: strptr("sales_order"), TopicResourceId: strptr("so_9"),
		AssigneeResourceType: strptr(string(constants.ObjectTypeAccountUser)), AssigneeResourceId: strptr("acus_9"), GroupId: strptr("mgrp_9"),
		CreatedAt: now, UpdatedAt: now,
		Participants: []*pb.ParticipantInfo{
			{Id: "cvpt_1", ConversationId: "cv_1", AccountId: "ac_1", ParticipantType: "user",
				AccountUserId: strptr("acus_1"), Role: "owner", Membership: "active", Notifications: "muted"},
		},
		LastMessage: &pb.MessageInfo{Id: "mg_last", ConversationId: "cv_1", AccountId: "ac_1", Sequence: 2, Kind: "chat",
			SenderAccountUserId: strptr("acus_1"), Body: strptr("hi"), CreatedAt: now, UpdatedAt: now},
	}
	c := ConversationFromProto(info)
	StashConversationMeta(ctx, info, &c)
	meta := resourcekit.GetLoadMeta(ctx)

	av, ok := meta.Get(constants.ObjectTypeConversation, "cv_1", "assignee")
	require.True(t, ok)
	assert.Equal(t, "acus_9", av.(*apiresource.Actor).ID)
	assert.Equal(t, constants.ActorTypeUser, av.(*apiresource.Actor).Type)

	gv, ok := meta.Get(constants.ObjectTypeConversation, "cv_1", "group")
	require.True(t, ok)
	assert.Equal(t, "mgrp_9", gv.(*apiresource.MessagingGroup).ID)
	assert.Equal(t, constants.ObjectTypeMessagingGroup, gv.(*apiresource.MessagingGroup).Object)

	pv, ok := meta.Get(constants.ObjectTypeConversation, "cv_1", "participants")
	require.True(t, ok)
	plist := pv.(*apiresource.List[apiresource.ConversationParticipant])
	require.Len(t, plist.Data, 1)
	p := plist.Data[0]
	assert.Equal(t, constants.ParticipantRoleOwner, p.Role)
	assert.Equal(t, constants.ParticipantNotificationsMuted, p.Notifications)
	require.NotNil(t, p.Actor)
	assert.Equal(t, "acus_1", p.Actor.ID)
	assert.Equal(t, constants.ActorTypeUser, p.Actor.Type)

	tv, ok := meta.Get(constants.ObjectTypeConversation, "cv_1", "topic")
	require.True(t, ok)
	assert.Equal(t, "so_9", tv.(*apiresource.Entity).ID)

	lv, ok := meta.Get(constants.ObjectTypeConversation, "cv_1", "last_message")
	require.True(t, ok)
	assert.Equal(t, "mg_last", lv.(*apiresource.Message).ID)
	// the last message's own sub-objects are stashed too
	sv, ok := meta.Get(constants.ObjectTypeChatMessage, "mg_last", "sender")
	require.True(t, ok)
	assert.Equal(t, "acus_1", sv.(*apiresource.Actor).ID)
}

func TestMessageFromProto_BaseLeavesExpandablesNil(t *testing.T) {
	now := timestamppb.Now()
	info := &pb.MessageInfo{
		Id: "mg_1", ConversationId: "cv_1", AccountId: "ac_1", Sequence: 7, Kind: "chat",
		SenderAccountUserId: strptr("acus_1"), ClientMessageId: strptr("cmid_1"),
		Body: strptr("hi"), LinkResourceType: strptr("sales_order"), LinkResourceId: strptr("so_1"),
		ReplyToMessageId: strptr("mg_0"), EditedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	m := MessageFromProto(info)
	assert.Equal(t, "mg_1", m.ID)
	assert.Equal(t, constants.ObjectTypeChatMessage, m.Object)
	assert.Equal(t, int64(7), m.Sequence)
	assert.Equal(t, constants.MessageKind("chat"), m.Kind)
	require.NotNil(t, m.EditedAt)
	assert.Nil(t, m.Conversation, "conversation is expandable — nil until resolved")
	assert.Nil(t, m.Sender, "sender is expandable — nil until resolved")
	assert.Nil(t, m.Author, "author is expandable — nil until resolved")
	assert.Nil(t, m.Resource, "resource is expandable — nil until resolved")
	assert.Nil(t, m.ReplyTo, "reply_to is expandable — nil until resolved")
	assert.Nil(t, m.Attachments, "attachments are expandable — nil until resolved")
}

func TestStashMessageMeta_FullMapping(t *testing.T) {
	ctx := metaCtx()
	now := timestamppb.Now()
	info := &pb.MessageInfo{
		Id: "mg_1", ConversationId: "cv_1", AccountId: "ac_1", Sequence: 7, Kind: "chat",
		SenderAccountUserId: strptr("acus_1"), Body: strptr("hi"),
		LinkResourceType: strptr("sales_order"), LinkResourceId: strptr("so_1"),
		ReplyToMessageId: strptr("mg_0"), CreatedAt: now, UpdatedAt: now,
	}
	m := MessageFromProto(info)
	StashMessageMeta(ctx, info, &m)
	meta := resourcekit.GetLoadMeta(ctx)
	ot := constants.ObjectTypeChatMessage

	sv, ok := meta.Get(ot, "mg_1", "sender")
	require.True(t, ok)
	sender := sv.(*apiresource.Actor)
	assert.Equal(t, constants.ActorTypeUser, sender.Type)
	assert.Equal(t, "acus_1", sender.ID)
	assert.Equal(t, constants.ObjectTypeActor, sender.Object)

	av, ok := meta.Get(ot, "mg_1", "author")
	require.True(t, ok)
	assert.Equal(t, "acus_1", av.(*apiresource.Actor).ID)
	assert.Equal(t, constants.ActorTypeUser, av.(*apiresource.Actor).Type)

	rv, ok := meta.Get(ot, "mg_1", "resource")
	require.True(t, ok)
	assert.Equal(t, "so_1", rv.(*apiresource.Entity).ID)

	cid, ok := meta.GetString(ot, "mg_1", "conversation_id")
	require.True(t, ok)
	assert.Equal(t, "cv_1", cid)

	rid, ok := meta.GetString(ot, "mg_1", "reply_to_id")
	require.True(t, ok)
	assert.Equal(t, "mg_0", rid)
}

func TestStashMessageMeta_NilSenderAndResource(t *testing.T) {
	ctx := metaCtx()
	now := timestamppb.Now()
	info := &pb.MessageInfo{
		Id: "mg_x", ConversationId: "cv_1", AccountId: "ac_1", Sequence: 1, Kind: "chat",
		DeletedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	m := MessageFromProto(info)
	StashMessageMeta(ctx, info, &m)
	meta := resourcekit.GetLoadMeta(ctx)
	ot := constants.ObjectTypeChatMessage

	_, ok := meta.Get(ot, "mg_x", "sender")
	assert.False(t, ok, "no sender id → no stashed sender")
	_, ok = meta.Get(ot, "mg_x", "resource")
	assert.False(t, ok, "no link → no stashed resource")
	_, ok = meta.Get(ot, "mg_x", "reply_to_id")
	assert.False(t, ok)
}

func TestStashMessageMeta_AliasSender(t *testing.T) {
	ctx := metaCtx()
	now := timestamppb.Now()
	// A customer-viewer payload: the notification service has already collapsed the staff author to the
	// branded "Customer Service" alias and cleared the real author, so only sender_alias_name is set.
	info := &pb.MessageInfo{
		Id: "mg_a", ConversationId: "cv_1", AccountId: "ac_1", Sequence: 4, Kind: "chat",
		Body: strptr("on it"), SenderAliasName: strptr("Customer Service"), CreatedAt: now, UpdatedAt: now,
	}
	m := MessageFromProto(info)
	StashMessageMeta(ctx, info, &m)
	meta := resourcekit.GetLoadMeta(ctx)
	ot := constants.ObjectTypeChatMessage

	sv, _ := meta.Get(ot, "mg_a", "sender")
	sender := sv.(*apiresource.Actor)
	assert.Equal(t, constants.ActorTypeGroup, sender.Type, "the displayed sender is the branded party, as a group actor")
	require.NotNil(t, sender.Name)
	assert.Equal(t, "Customer Service", *sender.Name)
	_, ok := meta.Get(ot, "mg_a", "author")
	assert.False(t, ok, "anonymized: no real author leaked")
}

func TestStashMessageMeta_Attachments(t *testing.T) {
	ctx := metaCtx()
	now := timestamppb.Now()
	info := &pb.MessageInfo{
		Id: "mg_att", ConversationId: "cv_1", AccountId: "ac_1", Sequence: 9, Kind: "chat",
		Body: strptr("see attached"), CreatedAt: now, UpdatedAt: now,
		Attachments: []*pb.AttachmentInfo{
			{Id: "mgah_1", Kind: "image", Url: strptr("https://s3/get?sig"), Filename: strptr("a.png"),
				ContentType: strptr("image/png"), CreatedAt: now},
			{Id: "mgah_2", Kind: "resource", ResourceType: strptr("sales_order"), ResourceId: strptr("so_1"), CreatedAt: now},
		},
	}
	m := MessageFromProto(info)
	StashMessageMeta(ctx, info, &m)
	av, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeChatMessage, "mg_att", "attachments")
	require.True(t, ok)
	list := av.(*apiresource.List[apiresource.MessageAttachment])
	require.Len(t, list.Data, 2)
	img := list.Data[0]
	assert.Equal(t, constants.MessageAttachmentKind("image"), img.Kind)
	require.NotNil(t, img.URL)
	assert.Nil(t, img.Resource)
	res := list.Data[1]
	// resource is expandable: nil on the base attachment, stashed for ?include=attachments.resource.
	assert.Nil(t, res.Resource)
	rv, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeMessageAttachment, "mgah_2", "resource")
	require.True(t, ok)
	assert.Equal(t, "so_1", rv.(*apiresource.Entity).ID)
	assert.Nil(t, res.URL)
}

func TestMessageFromProto_ScheduledAndDraftFields(t *testing.T) {
	now := timestamppb.Now()
	scheduled := MessageFromProto(&pb.MessageInfo{
		Id: "mg_1", ConversationId: "cv_1", AccountId: "ac_1", Kind: "chat",
		Status: "scheduled", Visibility: "internal", Body: strptr("later"),
		ScheduledFor: now, CreatedAt: now, UpdatedAt: now,
	})
	assert.Equal(t, constants.MessageStatusScheduled, scheduled.Status)
	require.NotNil(t, scheduled.ScheduledAt)

	draft := MessageFromProto(&pb.MessageInfo{
		Id: "mg_2", ConversationId: "cv_1", AccountId: "ac_1", Kind: "chat",
		Status: "draft", Visibility: "internal", Body: strptr("hi"),
		Channel: strptr("email"), Subject: strptr("Re: order"), CreatedAt: now, UpdatedAt: now,
	})
	assert.Equal(t, constants.MessageStatusDraft, draft.Status)
	assert.Equal(t, constants.MessageChannelEmail, draft.Channel)
	require.NotNil(t, draft.Subject)
	assert.Equal(t, "Re: order", *draft.Subject)
}

func TestMessageFromProto_DefaultsStatusSent(t *testing.T) {
	now := timestamppb.Now()
	m := MessageFromProto(&pb.MessageInfo{
		Id: "mg_3", ConversationId: "cv_1", AccountId: "ac_1", Kind: "chat", Visibility: "internal",
		CreatedAt: now, UpdatedAt: now,
	})
	assert.Equal(t, constants.MessageStatusSent, m.Status)
}

func TestMessageFromProto_Nil(t *testing.T) {
	assert.Equal(t, apiresource.Message{}, MessageFromProto(nil))
}

func TestConversationFromProto_Nil(t *testing.T) {
	assert.Equal(t, apiresource.Conversation{}, ConversationFromProto(nil))
}
