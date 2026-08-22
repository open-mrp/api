package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/notification"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type chatGRPCHandler struct {
	pb.UnimplementedChatServiceServer
	chatSvc domain.ConversationSvc
}

// NewChatGRPCHandler registers the ChatService (conversations + messages) handler.
func NewChatGRPCHandler(server *grpc.Server, chatSvc domain.ConversationSvc) *chatGRPCHandler {
	handler := &chatGRPCHandler{chatSvc: chatSvc}
	pb.RegisterChatServiceServer(server, handler)
	return handler
}

func (h *chatGRPCHandler) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	conv, apiErr := h.chatSvc.CreateConversation(ctx, domain.CreateConversationInput{
		Type:                      req.Type,
		Title:                     req.Title,
		GroupID:                   req.GroupId,
		TopicResourceType:         req.TopicResourceType,
		TopicResourceID:           req.TopicResourceId,
		ParticipantAccountUserIDs: req.ParticipantAccountUserIds,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) ListConversations(ctx context.Context, req *pb.ListConversationsRequest) (*pb.ListConversationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.chatSvc.ListConversations(ctx, domain.ListConversationsInput{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Type:   req.Type,
		Status: req.Status,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	conversations := make([]*pb.ConversationInfo, 0, len(page.Conversations))
	for _, c := range page.Conversations {
		conversations = append(conversations, conversationToProto(c))
	}
	return &pb.ListConversationsResponse{
		Conversations: conversations,
		PageInfo:      &pb.PageInfo{NextCursor: page.NextCursor, HasNextPage: page.HasNextPage},
	}, nil
}

func (h *chatGRPCHandler) GetConversation(ctx context.Context, req *pb.GetConversationRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.GetConversation(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) BatchGetConversations(ctx context.Context, req *pb.BatchGetConversationsRequest) (*pb.BatchGetConversationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	convs, apiErr := h.chatSvc.BatchGetConversations(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.ConversationInfo, 0, len(convs))
	for _, c := range convs {
		out = append(out, conversationToProto(c))
	}
	return &pb.BatchGetConversationsResponse{Conversations: out}, nil
}

func (h *chatGRPCHandler) ContactSupport(ctx context.Context, req *pb.ContactSupportRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	conv, apiErr := h.chatSvc.ContactSupport(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) GetSupportAvailability(ctx context.Context, req *pb.GetSupportAvailabilityRequest) (*pb.SupportAvailabilityInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	available, apiErr := h.chatSvc.SupportAvailability(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SupportAvailabilityInfo{Available: available}, nil
}

func (h *chatGRPCHandler) SetSupportRoute(ctx context.Context, req *pb.SetSupportRouteRequest) (*pb.SupportRouteInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	route, apiErr := h.chatSvc.SetSupportRoute(ctx, req.RelationAccountId, req.GroupConversationId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return supportRouteToProto(route), nil
}

func (h *chatGRPCHandler) ClearSupportRoute(ctx context.Context, req *pb.ClearSupportRouteRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.ClearSupportRoute(ctx, req.RelationAccountId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) GetSupportRoute(ctx context.Context, req *pb.GetSupportRouteRequest) (*pb.SupportRouteInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	route, apiErr := h.chatSvc.GetSupportRoute(ctx, req.RelationAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return supportRouteToProto(route), nil
}

func supportRouteToProto(route *domain.SupportRoute) *pb.SupportRouteInfo {
	return &pb.SupportRouteInfo{
		Id:                  route.ID,
		AccountId:           route.AccountID,
		RelationAccountId:   route.RelationAccountID,
		GroupConversationId: route.GroupConversationID,
		CreatedAt:           timestamppb.New(route.CreatedAt),
		UpdatedAt:           timestamppb.New(route.UpdatedAt),
	}
}

func (h *chatGRPCHandler) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	msg, apiErr := h.chatSvc.SendMessage(ctx, domain.SendMessageInput{
		ConversationID:        req.ConversationId,
		Visibility:            req.GetVisibility(),
		Body:                  req.Body,
		ClientMessageID:       req.ClientMessageId,
		ReplyToMessageID:      req.ReplyToMessageId,
		LinkResourceType:      req.LinkResourceType,
		LinkResourceID:        req.LinkResourceId,
		Subject:               req.Subject,
		Cc:                    req.Cc,
		Attachments:           attachmentInputsFromProto(req.Attachments),
		MentionAccountUserIDs: req.MentionAccountUserIds,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(msg), nil
}

func (h *chatGRPCHandler) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.chatSvc.ListMessages(ctx, domain.ListMessagesInput{
		ConversationID: req.ConversationId,
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		AfterSequence:  req.AfterSequence,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	messages := make([]*pb.MessageInfo, 0, len(page.Messages))
	for _, m := range page.Messages {
		messages = append(messages, messageToProto(m))
	}
	return &pb.ListMessagesResponse{
		Messages: messages,
		PageInfo: &pb.PageInfo{NextCursor: page.NextCursor, HasNextPage: page.HasNextPage},
	}, nil
}

func (h *chatGRPCHandler) BatchGetMessages(ctx context.Context, req *pb.BatchGetMessagesRequest) (*pb.BatchGetMessagesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	msgs, apiErr := h.chatSvc.BatchGetMessages(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.MessageInfo, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, messageToProto(m))
	}
	return &pb.BatchGetMessagesResponse{Messages: out}, nil
}

func (h *chatGRPCHandler) MarkConversationRead(ctx context.Context, req *pb.MarkConversationReadRequest) (*pb.MarkConversationReadResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	unread, apiErr := h.chatSvc.MarkConversationRead(ctx, req.ConversationId, req.UpToSequence)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.MarkConversationReadResponse{UnreadCount: unread}, nil
}

func (h *chatGRPCHandler) IsParticipant(ctx context.Context, req *pb.IsParticipantRequest) (*pb.IsParticipantResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ok, apiErr := h.chatSvc.IsParticipant(ctx, req.ConversationId, req.UserId, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.IsParticipantResponse{IsParticipant: ok}, nil
}

func (h *chatGRPCHandler) SendTyping(ctx context.Context, req *pb.SendTypingRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.SendTyping(ctx, req.ConversationId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) UpdateConversation(ctx context.Context, req *pb.UpdateConversationRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.UpdateConversation(ctx, req.ConversationId, req.Title, req.Status, req.ClearTitle)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) AddParticipant(ctx context.Context, req *pb.AddParticipantRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	role := ""
	if req.Role != nil {
		role = *req.Role
	}
	conv, apiErr := h.chatSvc.AddParticipant(ctx, req.ConversationId, req.AccountUserId, role)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.RemoveParticipant(ctx, req.ConversationId, req.ParticipantId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) UpdateParticipantRole(ctx context.Context, req *pb.UpdateParticipantRoleRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.UpdateParticipantRole(ctx, req.ConversationId, req.ParticipantId, req.Role)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) LeaveConversation(ctx context.Context, req *pb.LeaveConversationRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.Leave(ctx, req.ConversationId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) HideConversation(ctx context.Context, req *pb.HideConversationRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.Hide(ctx, req.ConversationId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) UnhideConversation(ctx context.Context, req *pb.UnhideConversationRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.Unhide(ctx, req.ConversationId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) SetMute(ctx context.Context, req *pb.SetMuteRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	var mutedUntil *time.Time
	if req.MutedUntil != nil {
		t := req.MutedUntil.AsTime()
		mutedUntil = &t
	}
	conv, apiErr := h.chatSvc.SetMute(ctx, req.ConversationId, req.Muted, mutedUntil)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) Block(ctx context.Context, req *pb.BlockRequest) (*pb.BlockInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	block, apiErr := h.chatSvc.Block(ctx, req.BlockedAccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BlockInfo{
		Id:                   block.ID,
		AccountId:            block.AccountID,
		BlockerAccountUserId: block.BlockerAccountUserID,
		BlockedAccountUserId: block.BlockedAccountUserID,
		CreatedAt:            timestamppb.New(block.CreatedAt),
	}, nil
}

func (h *chatGRPCHandler) Unblock(ctx context.Context, req *pb.UnblockRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.Unblock(ctx, req.BlockedAccountUserId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) ListBlocks(ctx context.Context, _ *pb.ListBlocksRequest) (*pb.ListBlocksResponse, error) {
	blocks, apiErr := h.chatSvc.ListBlocks(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.BlockInfo, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, &pb.BlockInfo{
			Id:                   b.ID,
			AccountId:            b.AccountID,
			BlockerAccountUserId: b.BlockerAccountUserID,
			BlockedAccountUserId: b.BlockedAccountUserID,
			CreatedAt:            timestamppb.New(b.CreatedAt),
		})
	}
	return &pb.ListBlocksResponse{Blocks: out}, nil
}

func (h *chatGRPCHandler) ListContacts(ctx context.Context, req *pb.ListContactsRequest) (*pb.ListContactsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	contacts, apiErr := h.chatSvc.ListContacts(ctx, req.Query)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.ContactInfo, 0, len(contacts))
	for _, c := range contacts {
		info := &pb.ContactInfo{
			Type: c.Type,
			Name: c.Name,
		}
		if c.AccountUserID != "" {
			acus := c.AccountUserID
			info.AccountUserId = &acus
		}
		out = append(out, info)
	}
	return &pb.ListContactsResponse{Contacts: out}, nil
}

func (h *chatGRPCHandler) ReportConversation(ctx context.Context, req *pb.ReportConversationRequest) (*pb.MessageReportInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	report, apiErr := h.chatSvc.ReportConversation(ctx, req.ConversationId, req.MessageId, req.Reason)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageReportToProto(report), nil
}

func messageReportToProto(r *domain.MessageReport) *pb.MessageReportInfo {
	info := &pb.MessageReportInfo{
		Id:             r.ID,
		ConversationId: r.ConversationID,
		Reason:         r.Reason,
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}
	if r.MessageID != "" {
		msgID := r.MessageID
		info.MessageId = &msgID
	}
	return info
}

// ── Reusable groups (rosters) ──

func (h *chatGRPCHandler) CreateMessagingGroup(ctx context.Context, req *pb.CreateMessagingGroupRequest) (*pb.MessagingGroupInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	group, apiErr := h.chatSvc.CreateMessagingGroup(ctx, domain.CreateMessagingGroupInput{
		Name:                 req.Name,
		MemberAccountUserIDs: req.MemberAccountUserIds,
		MemberAgentConfigIDs: req.MemberAgentConfigIds,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messagingGroupToProto(group), nil
}

func (h *chatGRPCHandler) ListMessagingGroups(ctx context.Context, req *pb.ListMessagingGroupsRequest) (*pb.ListMessagingGroupsResponse, error) {
	groups, apiErr := h.chatSvc.ListMessagingGroups(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.MessagingGroupInfo, 0, len(groups))
	for _, g := range groups {
		out = append(out, messagingGroupToProto(g))
	}
	return &pb.ListMessagingGroupsResponse{Groups: out}, nil
}

func (h *chatGRPCHandler) GetMessagingGroup(ctx context.Context, req *pb.GetMessagingGroupRequest) (*pb.MessagingGroupInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	group, apiErr := h.chatSvc.GetMessagingGroup(ctx, req.GroupId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messagingGroupToProto(group), nil
}

func (h *chatGRPCHandler) UpdateMessagingGroup(ctx context.Context, req *pb.UpdateMessagingGroupRequest) (*pb.MessagingGroupInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	group, apiErr := h.chatSvc.UpdateMessagingGroup(ctx, req.GroupId, req.Name)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messagingGroupToProto(group), nil
}

func (h *chatGRPCHandler) DeleteMessagingGroup(ctx context.Context, req *pb.DeleteMessagingGroupRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.DeleteMessagingGroup(ctx, req.GroupId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) AddMessagingGroupMember(ctx context.Context, req *pb.AddMessagingGroupMemberRequest) (*pb.MessagingGroupInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	input := domain.AddMessagingGroupMemberInput{
		GroupID:    req.GroupId,
		MemberType: req.MemberType,
	}
	if req.AccountUserId != nil {
		input.AccountUserID = *req.AccountUserId
	}
	if req.AgentConfigId != nil {
		input.AgentConfigID = *req.AgentConfigId
	}
	group, apiErr := h.chatSvc.AddMessagingGroupMember(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messagingGroupToProto(group), nil
}

func (h *chatGRPCHandler) RemoveMessagingGroupMember(ctx context.Context, req *pb.RemoveMessagingGroupMemberRequest) (*pb.MessagingGroupInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	group, apiErr := h.chatSvc.RemoveMessagingGroupMember(ctx, req.GroupId, req.MemberId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messagingGroupToProto(group), nil
}

func messagingGroupToProto(g *domain.MessagingGroup) *pb.MessagingGroupInfo {
	info := &pb.MessagingGroupInfo{
		Id:                     g.ID,
		AccountId:              g.AccountID,
		Name:                   g.Name,
		CreatedByAccountUserId: g.CreatedByAccountUserID,
		CreatedAt:              timestamppb.New(g.CreatedAt),
		UpdatedAt:              timestamppb.New(g.UpdatedAt),
	}
	for _, m := range g.Members {
		info.Members = append(info.Members, &pb.MessagingGroupMemberInfo{
			Id:            m.ID,
			GroupId:       m.GroupID,
			MemberType:    m.MemberType,
			AccountUserId: m.AccountUserID,
			AgentConfigId: m.AgentConfigID,
			CreatedAt:     timestamppb.New(m.CreatedAt),
		})
	}
	return info
}

// conversationStatusString maps the backing is_archived flag to the lifecycle status enum value.
func conversationStatusString(isArchived bool) string {
	if isArchived {
		return string(constants.ConversationStatusArchived)
	}
	return string(constants.ConversationStatusActive)
}

func conversationToProto(c *domain.Conversation) *pb.ConversationInfo {
	info := &pb.ConversationInfo{
		Id:                   c.ID,
		AccountId:            c.AccountID,
		Type:                 c.Type,
		Title:                c.Title,
		GroupId:              c.GroupID,
		TopicResourceType:    c.TopicResourceType,
		TopicResourceId:      c.TopicResourceID,
		LastMessageId:        c.LastMessageID,
		Status:               conversationStatusString(c.IsArchived),
		Unread:               c.Unread,
		Hidden:               c.Hidden,
		LegalHold:            c.LegalHold,
		Audience:             c.Audience,
		WorkflowStatus:       c.WorkflowStatus,
		AssigneeResourceType: c.AssigneeResourceType,
		AssigneeResourceId:   c.AssigneeResourceID,
		EmailInboxId:         c.EmailInboxID,
		CreatedAt:            timestamppb.New(c.CreatedAt),
		UpdatedAt:            timestamppb.New(c.UpdatedAt),
	}
	if c.LastMessageAt != nil {
		info.LastMessageAt = timestamppb.New(*c.LastMessageAt)
	}
	if c.LastMessage != nil {
		info.LastMessage = messageToProto(c.LastMessage)
	}
	for _, p := range c.Participants {
		info.Participants = append(info.Participants, participantToProto(p))
	}
	return info
}

func conversationLinkToProto(l *domain.ConversationLink) *pb.ConversationLinkInfo {
	return &pb.ConversationLinkInfo{
		Id:                     l.ID,
		AccountId:              l.AccountID,
		ConversationId:         l.ConversationID,
		ResourceType:           l.ResourceType,
		ResourceId:             l.ResourceID,
		CreatedByParticipantId: l.CreatedByParticipantID,
		CreatedAt:              timestamppb.New(l.CreatedAt),
	}
}

func participantToProto(p *domain.ConversationParticipant) *pb.ParticipantInfo {
	info := &pb.ParticipantInfo{
		Id:                     p.ID,
		ConversationId:         p.ConversationID,
		AccountId:              p.AccountID,
		ParticipantType:        p.ParticipantType,
		AccountUserId:          p.AccountUserID,
		AgentConfigId:          p.AgentConfigID,
		Role:                   p.Role,
		Membership:             p.Membership,
		Notifications:          p.Notifications,
		LastReadSequence:       p.LastReadSequence,
		LastReadMessageId:      p.LastReadMessageID,
		CreatedAt:              timestamppb.New(p.CreatedAt),
		UpdatedAt:              timestamppb.New(p.UpdatedAt),
		AgentTriggerPolicy:     p.AgentTriggerPolicy,
		AgentTriggerKeywords:   p.AgentTriggerKeywords,
		RelationAccountId:      p.RelationAccountID,
		AccountUserDisplayName: p.AccountUserDisplayName,
	}
	if p.LastReadAt != nil {
		info.LastReadAt = timestamppb.New(*p.LastReadAt)
	}
	return info
}

// agentFailureFromMetadata reads the agent-reply failure marker + error code recorded on a message's metadata (see agentFailureMetadata in the conversation service), so the client can flag a failed reply and react to a specific code.
func agentFailureFromMetadata(metadata json.RawMessage) (bool, *string) {
	if len(metadata) == 0 {
		return false, nil
	}
	var meta struct {
		AgentRunFailed bool   `json:"agent_run_failed"`
		ErrorCode      string `json:"error_code"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil || !meta.AgentRunFailed {
		return false, nil
	}
	if meta.ErrorCode == "" {
		return true, nil
	}
	return true, &meta.ErrorCode
}

func messageToProto(m *domain.Message) *pb.MessageInfo {
	info := &pb.MessageInfo{
		Id:                      m.ID,
		ConversationId:          m.ConversationID,
		AccountId:               m.AccountID,
		Sequence:                m.Sequence,
		Kind:                    m.Kind,
		Status:                  m.Status,
		Visibility:              m.Visibility,
		SenderParticipantId:     m.SenderParticipantID,
		SenderAccountUserId:     m.SenderAccountUserID,
		SenderAgentConfigId:     m.SenderAgentConfigID,
		AgentRunId:              m.AgentRunID,
		StreamingState:          m.StreamingState,
		ClientMessageId:         m.ClientMessageID,
		Body:                    m.Body,
		Preview:                 m.Preview,
		Channel:                 m.Channel,
		Subject:                 m.Subject,
		SourceThreadMessageId:   m.SourceThreadMessageID,
		ApprovedByAccountUserId: m.ApprovedByAccountUserID,
		LinkResourceType:        m.LinkResourceType,
		LinkResourceId:          m.LinkResourceID,
		ReplyToMessageId:        m.ReplyToMessageID,
		CreatedAt:               timestamppb.New(m.CreatedAt),
		UpdatedAt:               timestamppb.New(m.UpdatedAt),
		SenderAliasName:         m.SenderAlias,
		SenderDisplayName:       m.SenderDisplayName,
	}
	if failed, code := agentFailureFromMetadata(m.Metadata); failed {
		info.AgentRunFailed = true
		info.AgentErrorCode = code
	}
	if m.ScheduledFor != nil {
		info.ScheduledFor = timestamppb.New(*m.ScheduledFor)
	}
	if m.EditedAt != nil {
		info.EditedAt = timestamppb.New(*m.EditedAt)
	}
	if m.DeletedAt != nil {
		info.DeletedAt = timestamppb.New(*m.DeletedAt)
	}
	for _, a := range m.Attachments {
		info.Attachments = append(info.Attachments, attachmentToProto(a))
	}
	return info
}

func attachmentInputsFromProto(inputs []*pb.AttachmentInput) []domain.AttachmentInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]domain.AttachmentInput, 0, len(inputs))
	for _, in := range inputs {
		if in == nil {
			continue
		}
		out = append(out, domain.AttachmentInput{
			Kind:         in.Kind,
			S3Key:        in.S3Key,
			Filename:     in.Filename,
			ContentType:  in.ContentType,
			SizeBytes:    in.SizeBytes,
			URL:          in.Url,
			ResourceType: in.ResourceType,
			ResourceID:   in.ResourceId,
		})
	}
	return out
}

func attachmentToProto(a *domain.MessageAttachment) *pb.AttachmentInfo {
	return &pb.AttachmentInfo{
		Id:           a.ID,
		Kind:         a.Kind,
		Url:          a.URL,
		Filename:     a.Filename,
		ContentType:  a.ContentType,
		SizeBytes:    a.SizeBytes,
		ResourceType: a.ResourceType,
		ResourceId:   a.ResourceID,
		CreatedAt:    timestamppb.New(a.CreatedAt),
	}
}

func (h *chatGRPCHandler) ScheduleMessage(ctx context.Context, req *pb.ScheduleMessageRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var scheduledFor time.Time
	if req.ScheduledFor != nil {
		scheduledFor = req.ScheduledFor.AsTime()
	}
	sm, apiErr := h.chatSvc.ScheduleMessage(ctx, domain.CreateScheduledMessageInput{
		ConversationID: req.ConversationId,
		Body:           req.Body,
		ScheduledFor:   scheduledFor,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(sm), nil
}

func (h *chatGRPCHandler) ListScheduledMessages(ctx context.Context, req *pb.ListScheduledMessagesRequest) (*pb.ListScheduledMessagesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	items, apiErr := h.chatSvc.ListScheduledMessages(ctx, req.ConversationId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.MessageInfo, 0, len(items))
	for _, sm := range items {
		out = append(out, messageToProto(sm))
	}
	return &pb.ListScheduledMessagesResponse{ScheduledMessages: out}, nil
}

func (h *chatGRPCHandler) CancelScheduledMessage(ctx context.Context, req *pb.CancelScheduledMessageRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	sm, apiErr := h.chatSvc.CancelScheduledMessage(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(sm), nil
}

func (h *chatGRPCHandler) AddAgentParticipant(ctx context.Context, req *pb.AddAgentParticipantRequest) (*pb.ParticipantInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	policy := ""
	if req.TriggerPolicy != nil {
		policy = *req.TriggerPolicy
	}
	p, apiErr := h.chatSvc.AddAgentParticipant(ctx, domain.AddAgentParticipantInput{
		ConversationID:  req.ConversationId,
		AgentConfigID:   req.AgentConfigId,
		TriggerPolicy:   policy,
		TriggerKeywords: req.TriggerKeywords,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return participantToProto(p), nil
}

func (h *chatGRPCHandler) RemoveAgentParticipant(ctx context.Context, req *pb.RemoveAgentParticipantRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.RemoveAgentParticipant(ctx, req.ConversationId, req.ParticipantId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) SetLegalHold(ctx context.Context, req *pb.SetLegalHoldRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.SetLegalHold(ctx, req.ConversationId, req.LegalHold)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) RedactConversation(ctx context.Context, req *pb.RedactConversationRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.RedactConversation(ctx, req.ConversationId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) CreateAttachmentUploadURL(ctx context.Context, req *pb.CreateAttachmentUploadURLRequest) (*pb.AttachmentUploadTargetInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	contentType := ""
	if req.ContentType != nil {
		contentType = *req.ContentType
	}
	target, apiErr := h.chatSvc.CreateAttachmentUploadURL(ctx, req.ConversationId, req.Filename, contentType)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.AttachmentUploadTargetInfo{
		Attachment: attachmentToProto(target.Attachment),
		UploadUrl:  target.UploadURL,
		S3Key:      target.S3Key,
		ExpiresAt:  timestamppb.New(target.ExpiresAt),
	}, nil
}

// ── External customer-service cases ──

func (h *chatGRPCHandler) UpdateConversationWorkflow(ctx context.Context, req *pb.UpdateConversationWorkflowRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.UpdateWorkflowStatus(ctx, req.ConversationId, req.WorkflowStatus)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) AssignConversation(ctx context.Context, req *pb.AssignConversationRequest) (*pb.ConversationInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	conv, apiErr := h.chatSvc.AssignConversation(ctx, req.ConversationId, req.AssigneeResourceType, req.AssigneeResourceId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationToProto(conv), nil
}

func (h *chatGRPCHandler) ListInbox(ctx context.Context, req *pb.ListInboxRequest) (*pb.ListConversationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	page, apiErr := h.chatSvc.ListInbox(ctx, domain.SupportInboxInput{
		Cursor:             req.Cursor,
		Limit:              req.Limit,
		WorkflowStatus:     req.WorkflowStatus,
		AssigneeResourceID: req.AssigneeResourceId,
		Unassigned:         req.Unassigned,
		IncludeArchived:    req.IncludeArchived,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	conversations := make([]*pb.ConversationInfo, 0, len(page.Conversations))
	for _, c := range page.Conversations {
		conversations = append(conversations, conversationToProto(c))
	}
	return &pb.ListConversationsResponse{
		Conversations: conversations,
		PageInfo:      &pb.PageInfo{NextCursor: page.NextCursor, HasNextPage: page.HasNextPage},
	}, nil
}

func (h *chatGRPCHandler) ListConversationsByResource(ctx context.Context, req *pb.ListConversationsByResourceRequest) (*pb.ListConversationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	convs, apiErr := h.chatSvc.ListConversationsByResource(ctx, req.ResourceType, req.ResourceId, req.Limit)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.ConversationInfo, 0, len(convs))
	for _, c := range convs {
		out = append(out, conversationToProto(c))
	}
	return &pb.ListConversationsResponse{Conversations: out, PageInfo: &pb.PageInfo{}}, nil
}

func (h *chatGRPCHandler) AddConversationLink(ctx context.Context, req *pb.AddConversationLinkRequest) (*pb.ConversationLinkInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	link, apiErr := h.chatSvc.AddConversationLink(ctx, req.ConversationId, req.ResourceType, req.ResourceId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return conversationLinkToProto(link), nil
}

func (h *chatGRPCHandler) RemoveConversationLink(ctx context.Context, req *pb.RemoveConversationLinkRequest) (*pb.ChatAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.chatSvc.RemoveConversationLink(ctx, req.ConversationId, req.LinkId); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChatAck{}, nil
}

func (h *chatGRPCHandler) ListConversationLinks(ctx context.Context, req *pb.ListConversationLinksRequest) (*pb.ListConversationLinksResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	links, apiErr := h.chatSvc.ListConversationLinks(ctx, req.ConversationId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.ConversationLinkInfo, 0, len(links))
	for _, l := range links {
		out = append(out, conversationLinkToProto(l))
	}
	return &pb.ListConversationLinksResponse{Links: out}, nil
}

func (h *chatGRPCHandler) CreateReplyDraft(ctx context.Context, req *pb.CreateReplyDraftRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	draft, apiErr := h.chatSvc.CreateReplyDraft(ctx, domain.CreateReplyDraftInput{
		ConversationID:        req.ConversationId,
		Channel:               req.Channel,
		Body:                  req.Body,
		Subject:               req.Subject,
		SourceThreadMessageID: req.SourceThreadMessageId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(draft), nil
}

func (h *chatGRPCHandler) ListReplyDrafts(ctx context.Context, req *pb.ListReplyDraftsRequest) (*pb.ListReplyDraftsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	drafts, apiErr := h.chatSvc.ListReplyDrafts(ctx, req.ConversationId, req.Status)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.MessageInfo, 0, len(drafts))
	for _, d := range drafts {
		out = append(out, messageToProto(d))
	}
	return &pb.ListReplyDraftsResponse{Drafts: out}, nil
}

func (h *chatGRPCHandler) UpdateReplyDraft(ctx context.Context, req *pb.UpdateReplyDraftRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	draft, apiErr := h.chatSvc.UpdateReplyDraft(ctx, req.DraftId, req.Body, req.Subject)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(draft), nil
}

func (h *chatGRPCHandler) RejectReplyDraft(ctx context.Context, req *pb.RejectReplyDraftRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	draft, apiErr := h.chatSvc.RejectReplyDraft(ctx, req.DraftId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(draft), nil
}

func (h *chatGRPCHandler) ApproveAndSendReplyDraft(ctx context.Context, req *pb.ApproveAndSendReplyDraftRequest) (*pb.MessageInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()
	draft, apiErr := h.chatSvc.ApproveAndSendReplyDraft(ctx, req.DraftId, req.ClientMessageId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return messageToProto(draft), nil
}
