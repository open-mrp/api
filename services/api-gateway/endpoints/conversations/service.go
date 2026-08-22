package conversationep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConversationSvc backs the conversation endpoints via the notification-service ChatService gRPC client.
type ConversationSvc interface {
	CreateConversation(ctx context.Context, req *CreateConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	ListConversations(ctx context.Context, req *ListConversationsRequest) (*apiresource.List[apiresource.Conversation], *apierror.APIError)
	GetConversation(ctx context.Context, req *RetrieveConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	ContactSupport(ctx context.Context, req *ContactSupportRequest) (*apiresource.Conversation, *apierror.APIError)
	SupportAvailability(ctx context.Context, req *SupportAvailabilityRequest) (*apiresource.SupportAvailability, *apierror.APIError)
	MarkConversationRead(ctx context.Context, req *MarkConversationReadRequest) (*apiresource.Conversation, *apierror.APIError)
	UpdateConversation(ctx context.Context, req *UpdateConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	ArchiveConversation(ctx context.Context, req *ArchiveConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	UnarchiveConversation(ctx context.Context, req *UnarchiveConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	LeaveConversation(ctx context.Context, req *LeaveConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	HideConversation(ctx context.Context, req *HideConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	UnhideConversation(ctx context.Context, req *UnhideConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	MuteConversation(ctx context.Context, req *MuteConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	UnmuteConversation(ctx context.Context, req *UnmuteConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	SendTyping(ctx context.Context, req *TypingRequest) (*apiresource.MessageResource, *apierror.APIError)
	SetLegalHold(ctx context.Context, req *SetLegalHoldRequest) (*apiresource.Conversation, *apierror.APIError)
	RedactConversation(ctx context.Context, req *RedactConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	// External customer-service cases: triage, assignment, links, report.
	SetWorkflowStatus(ctx context.Context, req *SetWorkflowStatusRequest) (*apiresource.Conversation, *apierror.APIError)
	AssignConversation(ctx context.Context, req *AssignConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	ReportConversation(ctx context.Context, req *ReportConversationRequest) (*apiresource.Conversation, *apierror.APIError)
	AddConversationLink(ctx context.Context, req *AddConversationLinkRequest) (*apiresource.ConversationLink, *apierror.APIError)
	RemoveConversationLink(ctx context.Context, req *RemoveConversationLinkRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListConversationLinks(ctx context.Context, req *ListConversationLinksRequest) (*apiresource.List[apiresource.ConversationLink], *apierror.APIError)
}

type ConversationSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type conversationSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var conversationSvcTracer = tracing.GetTracer("api-gateway.endpoints.conversations.service")

func (c *ConversationSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("conversation endpoint service: chat client is required")
	}
	return nil
}

func NewConversationSvc(config *ConversationSvcConfig) ConversationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &conversationSvcImpl{chatClient: config.ChatClient}
}

// toResource maps a proto conversation to its base API resource, stashing the expandable sub-objects
// (assignee, group, participants, topic, last_message) for the include resolver and
// hydrating display names for the sub-objects the caller actually requested via ?include=.
func (s *conversationSvcImpl) toResource(ctx context.Context, c *pb.ConversationInfo) apiresource.Conversation {
	res := chatmap.ConversationFromProto(c)
	chatmap.StashConversationMeta(ctx, c, &res)
	s.hydrateConversations(ctx, &res)
	return res
}

// toResourceList maps proto conversations to base resources, stashing each one's expandables and
// hydrating requested sub-object names in a single batch across the whole page.
func (s *conversationSvcImpl) toResourceList(ctx context.Context, cs []*pb.ConversationInfo) []apiresource.Conversation {
	items := make([]apiresource.Conversation, len(cs))
	ptrs := make([]*apiresource.Conversation, len(cs))
	for i, c := range cs {
		items[i] = chatmap.ConversationFromProto(c)
		chatmap.StashConversationMeta(ctx, cs[i], &items[i])
		ptrs[i] = &items[i]
	}
	s.hydrateConversations(ctx, ptrs...)
	return items
}

// hydrateConversations fills display names on the expandable sub-objects the caller requested via
// ?include=, reading the stashed sub-objects from LoadMeta: the group roster when `group` is included,
// the assignee actor when `assignee` is included, participant actors when `participants` is included,
// and the last message's sender/author when those nested paths are included. The chat
// service stores actors and rosters by id only, so names are resolved here (users via core, agents via
// agent-service, rosters via the chat service). Best-effort and a no-op when nothing relevant was requested.
func (s *conversationSvcImpl) hydrateConversations(ctx context.Context, convs ...*apiresource.Conversation) {
	requested := resourcekit.RequestedIncludeSet(ctx)
	if len(requested) == 0 {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	var actors []*apiresource.Actor
	var groups []*apiresource.MessagingGroup
	var topics []*apiresource.Entity

	if requested["topic"] {
		for _, c := range convs {
			if v, ok := meta.Get(constants.ObjectTypeConversation, c.ID, "topic"); ok {
				topics = append(topics, v.(*apiresource.Entity))
			}
		}
	}

	if requested["group"] {
		for _, c := range convs {
			if v, ok := meta.Get(constants.ObjectTypeConversation, c.ID, "group"); ok {
				groups = append(groups, v.(*apiresource.MessagingGroup))
			}
		}
	}

	if requested["assignee"] {
		for _, c := range convs {
			if v, ok := meta.Get(constants.ObjectTypeConversation, c.ID, "assignee"); ok {
				actors = append(actors, v.(*apiresource.Actor))
			}
		}
	}

	if requested["participants"] {
		for _, c := range convs {
			if v, ok := meta.Get(constants.ObjectTypeConversation, c.ID, "participants"); ok {
				list := v.(*apiresource.List[apiresource.ConversationParticipant])
				for i := range list.Data {
					if a := list.Data[i].Actor; a != nil {
						actors = append(actors, a)
					}
				}
			}
		}
	}

	if requested["last_message.sender"] || requested["last_message.author"] {
		for _, c := range convs {
			v, ok := meta.Get(constants.ObjectTypeConversation, c.ID, "last_message")
			if !ok {
				continue
			}
			lm := v.(*apiresource.Message)
			if requested["last_message.sender"] {
				if sv, ok := meta.Get(constants.ObjectTypeChatMessage, lm.ID, "sender"); ok {
					actors = append(actors, sv.(*apiresource.Actor))
				}
			}
			if requested["last_message.author"] {
				if av, ok := meta.Get(constants.ObjectTypeChatMessage, lm.ID, "author"); ok {
					actors = append(actors, av.(*apiresource.Actor))
				}
			}
		}
	}

	resourceloaders.HydrateActorNames(ctx, actors)
	resourceloaders.HydrateMessagingGroups(ctx, groups)
	// A customer-facing case anchors its topic to the customer record, so the customer's name/number
	// come from the topic entity (no-op for non-customer topics, which stay bare id/type references).
	resourceloaders.HydrateCustomerEntities(ctx, topics)
}

func (s *conversationSvcImpl) CreateConversation(ctx context.Context, req *CreateConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	pbReq := &pb.CreateConversationRequest{
		Type:                      string(req.Type),
		ParticipantAccountUserIds: req.ParticipantAccountUserIDs,
		Title:                     req.Title.Ptr(),
	}
	if t, ok := req.TopicResourceType.Value(); ok {
		s := string(t)
		pbReq.TopicResourceType = &s
	}
	pbReq.TopicResourceId = req.TopicResourceID.Ptr()
	pbReq.GroupId = req.GroupID.Ptr()

	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.CreateConversation(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) ListConversations(ctx context.Context, req *ListConversationsRequest) (*apiresource.List[apiresource.Conversation], *apierror.APIError) {
	// Filter dispatch: a topic-resource anchor returns the record's discussions; any inbox filter
	// returns the external-case support inbox; otherwise the caller's normal conversation list.
	if req.isByRecordQuery() {
		return s.listByResource(ctx, req)
	}
	if req.isInboxQuery() {
		return s.listInbox(ctx, req)
	}

	status := string(constants.ConversationListStatusActive)
	if req.Status != nil {
		status = string(*req.Status)
	}
	pbReq := &pb.ListConversationsRequest{Limit: req.Limit, Cursor: req.Cursor, Status: status}
	if req.Type != nil {
		t := string(*req.Type)
		pbReq.Type = &t
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListConversationsResponse, error) {
			return s.chatClient.ListConversations(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	items := s.toResourceList(ctx, resp.Conversations)
	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

func (s *conversationSvcImpl) GetConversation(ctx context.Context, req *RetrieveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.GetConversation(ctx, &pb.GetConversationRequest{Id: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) ContactSupport(ctx context.Context, _ *ContactSupportRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.contact_support", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.ContactSupport(ctx, &pb.ContactSupportRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) SupportAvailability(ctx context.Context, _ *SupportAvailabilityRequest) (*apiresource.SupportAvailability, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.support_availability", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SupportAvailabilityInfo, error) {
			return s.chatClient.GetSupportAvailability(ctx, &pb.GetSupportAvailabilityRequest{}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.SupportAvailability{Object: constants.ObjectTypeSupportAvailability, Available: resp.Available}, nil
}

func (s *conversationSvcImpl) MarkConversationRead(ctx context.Context, req *MarkConversationReadRequest) (*apiresource.Conversation, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.mark_read", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MarkConversationReadResponse, error) {
			return s.chatClient.MarkConversationRead(ctx, &pb.MarkConversationReadRequest{
				ConversationId: req.ConversationID,
				UpToSequence:   req.UpToSequence,
			}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	// Return the refreshed conversation (with the new unread count).
	return s.GetConversation(ctx, &RetrieveConversationRequest{ConversationID: req.ConversationID})
}

func (s *conversationSvcImpl) UpdateConversation(ctx context.Context, req *UpdateConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	pbReq := &pb.UpdateConversationRequest{ConversationId: req.ConversationID, Title: req.Title.ValuePtr(), ClearTitle: req.Title.IsClear()}
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.UpdateConversation(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

// ArchiveConversation archives a conversation at the account level (owner/admin; groups only).
func (s *conversationSvcImpl) ArchiveConversation(ctx context.Context, req *ArchiveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	return s.setArchived(ctx, req.ConversationID, constants.ConversationStatusArchived)
}

// UnarchiveConversation returns an archived conversation to the active state.
func (s *conversationSvcImpl) UnarchiveConversation(ctx context.Context, req *UnarchiveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	return s.setArchived(ctx, req.ConversationID, constants.ConversationStatusActive)
}

func (s *conversationSvcImpl) setArchived(ctx context.Context, conversationID string, status constants.ConversationStatus) (*apiresource.Conversation, *apierror.APIError) {
	st := string(status)
	pbReq := &pb.UpdateConversationRequest{ConversationId: conversationID, Status: &st}
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.UpdateConversation(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) SetLegalHold(ctx context.Context, req *SetLegalHoldRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.set_legal_hold", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.SetLegalHold(ctx, &pb.SetLegalHoldRequest{ConversationId: req.ConversationID, LegalHold: req.LegalHold == constants.LegalHoldStatusHeld}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) RedactConversation(ctx context.Context, req *RedactConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.redact", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.RedactConversation(ctx, &pb.RedactConversationRequest{ConversationId: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}

func (s *conversationSvcImpl) LeaveConversation(ctx context.Context, req *LeaveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.leave", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.LeaveConversation(ctx, &pb.LeaveConversationRequest{ConversationId: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	// Leaving is staff-only and keeps the participant row (state='left'), so the caller can still read back
	// the conversation — now reflecting that they are no longer an active participant and it is hidden.
	return s.GetConversation(ctx, &RetrieveConversationRequest{ConversationID: req.ConversationID})
}

func (s *conversationSvcImpl) HideConversation(ctx context.Context, req *HideConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.hide", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.HideConversation(ctx, &pb.HideConversationRequest{ConversationId: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	// Return the refreshed conversation (now marked hidden for the caller).
	return s.GetConversation(ctx, &RetrieveConversationRequest{ConversationID: req.ConversationID})
}

func (s *conversationSvcImpl) UnhideConversation(ctx context.Context, req *UnhideConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.unhide", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.UnhideConversation(ctx, &pb.UnhideConversationRequest{ConversationId: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	// Return the refreshed conversation (no longer hidden for the caller).
	return s.GetConversation(ctx, &RetrieveConversationRequest{ConversationID: req.ConversationID})
}

func (s *conversationSvcImpl) SendTyping(ctx context.Context, req *TypingRequest) (*apiresource.MessageResource, *apierror.APIError) {
	_, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.typing", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChatAck, error) {
			return s.chatClient.SendTyping(ctx, &pb.SendTypingRequest{ConversationId: req.ConversationID}, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return chatmap.OkMessage("Typing."), nil
}

func (s *conversationSvcImpl) MuteConversation(ctx context.Context, req *MuteConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	pbReq := &pb.SetMuteRequest{ConversationId: req.ConversationID, Muted: true}
	if mu, ok := req.MutedUntil.Value(); ok {
		pbReq.MutedUntil = timestamppb.New(mu)
	}
	return s.setMute(ctx, pbReq)
}

func (s *conversationSvcImpl) UnmuteConversation(ctx context.Context, req *UnmuteConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
	return s.setMute(ctx, &pb.SetMuteRequest{ConversationId: req.ConversationID, Muted: false})
}

func (s *conversationSvcImpl) setMute(ctx context.Context, pbReq *pb.SetMuteRequest) (*apiresource.Conversation, *apierror.APIError) {
	resp, rpcErr := grpcutil.CallRPC(ctx, conversationSvcTracer, "service.conversations.set_mute", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConversationInfo, error) {
			return s.chatClient.SetMute(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := s.toResource(ctx, resp)
	return &result, nil
}
