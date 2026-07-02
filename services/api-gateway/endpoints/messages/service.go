package messageep

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/chatmap"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MessageSvc backs the chat message endpoints via the notification-service ChatService gRPC client. A
// message is the single resource for sent, scheduled, and draft content; the lifecycle endpoints
// (send, schedule, draft create/update/approve/reject/cancel) all return a Message.
type MessageSvc interface {
	ListMessages(ctx context.Context, req *ListMessagesRequest) (*apiresource.List[apiresource.Message], *apierror.APIError)
	SendMessage(ctx context.Context, req *SendMessageRequest) (*apiresource.Message, *apierror.APIError)
	UpdateDraft(ctx context.Context, req *UpdateDraftRequest) (*apiresource.Message, *apierror.APIError)
	ApproveSendDraft(ctx context.Context, req *ApproveSendDraftRequest) (*apiresource.Message, *apierror.APIError)
	RejectDraft(ctx context.Context, req *RejectDraftRequest) (*apiresource.Message, *apierror.APIError)
	CancelScheduled(ctx context.Context, req *CancelScheduledRequest) (*apiresource.Message, *apierror.APIError)
}

type MessageSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type messageSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var messageSvcTracer = tracing.GetTracer("api-gateway.endpoints.messages.service")

func (c *MessageSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("messages endpoint service: chat client is required")
	}
	return nil
}

func NewMessageSvc(config *MessageSvcConfig) MessageSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &messageSvcImpl{chatClient: config.ChatClient}
}

// resolveOne maps a single message proto to its resource, stashing expandable sub-objects and
// hydrating the requested ?include= names.
func (s *messageSvcImpl) resolveOne(ctx context.Context, m *pb.MessageInfo) *apiresource.Message {
	result := chatmap.MessageFromProto(m)
	chatmap.StashMessageMeta(ctx, m, &result)
	s.hydrateMessages(ctx, &result)
	return &result
}

func (s *messageSvcImpl) resolveList(ctx context.Context, msgs []*pb.MessageInfo) []apiresource.Message {
	items := make([]apiresource.Message, len(msgs))
	ptrs := make([]*apiresource.Message, len(msgs))
	for i, m := range msgs {
		items[i] = chatmap.MessageFromProto(m)
		chatmap.StashMessageMeta(ctx, m, &items[i])
		ptrs[i] = &items[i]
	}
	s.hydrateMessages(ctx, ptrs...)
	return items
}

// ListMessages returns a conversation's timeline (status=sent, default), its open customer-reply
// drafts (status=draft), or the caller's scheduled messages (status=scheduled) — dispatching to the
// appropriate RPC, all returning Message.
func (s *messageSvcImpl) ListMessages(ctx context.Context, req *ListMessagesRequest) (*apiresource.List[apiresource.Message], *apierror.APIError) {
	if req.Status != nil && *req.Status == constants.MessageStatusDraft {
		pbReq := &pb.ListReplyDraftsRequest{ConversationId: req.ConversationID}
		st := string(constants.MessageStatusDraft)
		pbReq.Status = &st
		resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.list_drafts", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListReplyDraftsResponse, error) {
				return s.chatClient.ListReplyDrafts(ctx, pbReq, opts...)
			})
		if rpcErr != nil {
			return nil, rpcErr
		}
		items := s.resolveList(ctx, resp.Drafts)
		return apiresource.NewList(items, apiresource.PageInfo{}), nil
	}

	if req.Status != nil && *req.Status == constants.MessageStatusScheduled {
		pbReq := &pb.ListScheduledMessagesRequest{ConversationId: req.ConversationID}
		resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.list_scheduled_messages", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListScheduledMessagesResponse, error) {
				return s.chatClient.ListScheduledMessages(ctx, pbReq, opts...)
			})
		if rpcErr != nil {
			return nil, rpcErr
		}
		items := s.resolveList(ctx, resp.ScheduledMessages)
		return apiresource.NewList(items, apiresource.PageInfo{}), nil
	}

	pbReq := &pb.ListMessagesRequest{
		ConversationId: req.ConversationID,
		Limit:          req.Limit,
		Cursor:         req.Cursor,
		AfterSequence:  req.AfterSequence,
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.list_messages", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMessagesResponse, error) {
			return s.chatClient.ListMessages(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	items := s.resolveList(ctx, resp.Messages)
	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

// SendMessage posts a message. With scheduled_at it queues the message for future delivery (status
// scheduled); with audience=customer it sends a customer-visible reply (branded "Customer Service",
// delivered by email on an email-bridged case).
func (s *messageSvcImpl) SendMessage(ctx context.Context, req *SendMessageRequest) (*apiresource.Message, *apierror.APIError) {
	// Draft mode: propose a customer-reply draft instead of sending. A draft mints a fresh id (no dedupe key) and is held for approval.
	if mode, _ := req.Mode.Value(); mode == constants.MessageSendModeDraft {
		channel, ok := req.Channel.Value()
		if !ok {
			return nil, apierror.NewParameterMissingError("A channel is required for a draft.", "channel")
		}
		pbReq := &pb.CreateReplyDraftRequest{
			ConversationId:        req.ConversationID,
			Channel:               string(channel),
			Body:                  req.Body,
			Subject:               req.Subject.Ptr(),
			SourceThreadMessageId: req.SourceThreadMessageID.Ptr(),
		}
		resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.create_draft", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
				return s.chatClient.CreateReplyDraft(ctx, pbReq, opts...)
			})
		if rpcErr != nil {
			return nil, rpcErr
		}
		return s.resolveOne(ctx, resp), nil
	}

	if req.ClientMessageID == "" {
		return nil, apierror.NewParameterMissingError("A client_message_id is required.", "client_message_id")
	}

	// Future delivery: dispatch to the scheduler. A scheduled message carries no immediate timeline slot.
	if at, ok := req.ScheduledAt.Value(); ok {
		pbReq := &pb.ScheduleMessageRequest{
			ConversationId: req.ConversationID,
			Body:           req.Body,
			ScheduledFor:   timestamppb.New(at),
		}
		resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.schedule_message", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
				return s.chatClient.ScheduleMessage(ctx, pbReq, opts...)
			})
		if rpcErr != nil {
			return nil, rpcErr
		}
		return s.resolveOne(ctx, resp), nil
	}

	pbReq := &pb.SendMessageRequest{
		ConversationId:        req.ConversationID,
		Body:                  &req.Body,
		ClientMessageId:       req.ClientMessageID,
		ReplyToMessageId:      req.ReplyToMessageID.Ptr(),
		LinkResourceId:        req.LinkResourceID.Ptr(),
		Subject:               req.Subject.Ptr(),
		Cc:                    req.Cc,
		Attachments:           attachmentInputsToProto(req.Attachments),
		MentionAccountUserIds: req.Mentions,
	}
	if lt, ok := req.LinkResourceType.Value(); ok {
		s := string(lt)
		pbReq.LinkResourceType = &s
	}
	// audience=customer maps to a customer-visible send; empty/internal leaves the default (internal).
	if aud, ok := req.Audience.Value(); ok && aud == constants.ConversationAudienceCustomer {
		v := string(constants.MessageVisibilityExternal)
		pbReq.Visibility = &v
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.send_message", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
			return s.chatClient.SendMessage(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.resolveOne(ctx, resp), nil
}

func (s *messageSvcImpl) UpdateDraft(ctx context.Context, req *UpdateDraftRequest) (*apiresource.Message, *apierror.APIError) {
	pbReq := &pb.UpdateReplyDraftRequest{DraftId: req.MessageID, Body: req.Body, Subject: req.Subject.Ptr()}
	resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.update_draft", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
			return s.chatClient.UpdateReplyDraft(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.resolveOne(ctx, resp), nil
}

func (s *messageSvcImpl) ApproveSendDraft(ctx context.Context, req *ApproveSendDraftRequest) (*apiresource.Message, *apierror.APIError) {
	pbReq := &pb.ApproveAndSendReplyDraftRequest{DraftId: req.MessageID, ClientMessageId: req.ClientMessageID}
	resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.approve_draft", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
			return s.chatClient.ApproveAndSendReplyDraft(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.resolveOne(ctx, resp), nil
}

func (s *messageSvcImpl) RejectDraft(ctx context.Context, req *RejectDraftRequest) (*apiresource.Message, *apierror.APIError) {
	pbReq := &pb.RejectReplyDraftRequest{DraftId: req.MessageID}
	resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.reject_draft", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
			return s.chatClient.RejectReplyDraft(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.resolveOne(ctx, resp), nil
}

func (s *messageSvcImpl) CancelScheduled(ctx context.Context, req *CancelScheduledRequest) (*apiresource.Message, *apierror.APIError) {
	pbReq := &pb.CancelScheduledMessageRequest{Id: req.MessageID}
	resp, rpcErr := grpcutil.CallRPC(ctx, messageSvcTracer, "service.conversations.cancel_scheduled", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MessageInfo, error) {
			return s.chatClient.CancelScheduledMessage(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return s.resolveOne(ctx, resp), nil
}

// hydrateMessages fills display names on the expandable message sub-objects the caller requested via
// ?include=, reading the stashed sub-objects from LoadMeta: sender/author actor names when those are
// included, and the linked resource's name when `resource` is included. Best-effort and a no-op when
// nothing relevant was requested.
func (s *messageSvcImpl) hydrateMessages(ctx context.Context, msgs ...*apiresource.Message) {
	requested := resourcekit.RequestedIncludeSet(ctx)
	if len(requested) == 0 {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	var actors []*apiresource.Actor
	if requested["sender"] {
		for _, m := range msgs {
			if v, ok := meta.Get(constants.ObjectTypeChatMessage, m.ID, "sender"); ok {
				actors = append(actors, v.(*apiresource.Actor))
			}
		}
	}
	if requested["author"] {
		for _, m := range msgs {
			if v, ok := meta.Get(constants.ObjectTypeChatMessage, m.ID, "author"); ok {
				actors = append(actors, v.(*apiresource.Actor))
			}
		}
	}
	resourceloaders.HydrateActorNames(ctx, actors)

	if requested["resource"] {
		var entities []*apiresource.Entity
		for _, m := range msgs {
			if v, ok := meta.Get(constants.ObjectTypeChatMessage, m.ID, "resource"); ok {
				entities = append(entities, v.(*apiresource.Entity))
			}
		}
		hydrateResourceEntityNames(ctx, entities)
	}
}

// hydrateResourceEntityNames fills the display name on linked-resource references (sales orders and
// invoices) by batch-resolving their ids. Other resource types are left as bare id/type references.
func hydrateResourceEntityNames(ctx context.Context, entities []*apiresource.Entity) {
	var salesOrderIDs, invoiceIDs []string
	seen := make(map[string]struct{})
	collect := func(id string, dst *[]string) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		*dst = append(*dst, id)
	}
	for _, e := range entities {
		if e == nil || e.Name != nil || e.ID == "" {
			continue
		}
		switch e.Type {
		case constants.ObjectTypeSalesOrder:
			collect(e.ID, &salesOrderIDs)
		case constants.ObjectTypeInvoice:
			collect(e.ID, &invoiceIDs)
		}
	}

	names := make(map[string]string)
	if len(salesOrderIDs) > 0 {
		if loaded, apiErr := resourceloaders.LoadSalesOrders(ctx, salesOrderIDs); apiErr == nil {
			for id, ref := range loaded {
				if so, ok := ref.(*apiresource.SalesOrder); ok && so.Number != "" {
					names[id] = "Order #" + formatRecordNumber(so.Number)
				}
			}
		}
	}
	if len(invoiceIDs) > 0 {
		if loaded, apiErr := resourceloaders.LoadInvoices(ctx, invoiceIDs); apiErr == nil {
			for id, ref := range loaded {
				if inv, ok := ref.(*apiresource.Invoice); ok && inv.Number != "" {
					names[id] = "Invoice #" + formatRecordNumber(inv.Number)
				}
			}
		}
	}

	for _, e := range entities {
		if e == nil || e.Name != nil {
			continue
		}
		if name, ok := names[e.ID]; ok {
			e.Name = &name
		}
	}
}

func formatRecordNumber(number string) string {
	parts := strings.Split(number, "-")
	if len(parts) > 0 && parts[0] != "" {
		c := parts[0][0]
		isLetter := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		if !isLetter {
			for len(parts[0]) < 6 {
				parts[0] = "0" + parts[0]
			}
		}
	}
	return strings.Join(parts, "-")
}

func attachmentInputsToProto(inputs []MessageAttachmentInput) []*pb.AttachmentInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]*pb.AttachmentInput, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, &pb.AttachmentInput{
			Kind:         string(in.Kind),
			S3Key:        in.S3Key.Ptr(),
			Filename:     in.Filename.Ptr(),
			ContentType:  in.ContentType.Ptr(),
			SizeBytes:    in.SizeBytes.Ptr(),
			Url:          in.URL.Ptr(),
			ResourceType: in.ResourceType.Ptr(),
			ResourceId:   in.ResourceID.Ptr(),
		})
	}
	return out
}
