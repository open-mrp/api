package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/notification"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type emailBridgeGRPCHandler struct {
	pb.UnimplementedEmailBridgeServiceServer
	emailBridgeSvc domain.EmailBridgeSvc
	chatSvc        domain.ConversationSvc
	// inboundEmailDomain is the Augno SES receiving subdomain used to render each inbox's forwarding
	// address (<inbox_id>@<domain>). Empty when the subdomain isn't configured → forwarding_address unset.
	inboundEmailDomain string
}

// NewEmailBridgeGRPCHandler registers the EmailBridgeService (domain verification + inbox CRUD + agent send/draft) handler. chatSvc backs the outbound send + draft RPCs. inboundEmailDomain renders per-inbox forwarding addresses (empty to omit them).
func NewEmailBridgeGRPCHandler(server *grpc.Server, emailBridgeSvc domain.EmailBridgeSvc, chatSvc domain.ConversationSvc, inboundEmailDomain string) *emailBridgeGRPCHandler {
	handler := &emailBridgeGRPCHandler{emailBridgeSvc: emailBridgeSvc, chatSvc: chatSvc, inboundEmailDomain: inboundEmailDomain}
	pb.RegisterEmailBridgeServiceServer(server, handler)
	return handler
}

func (h *emailBridgeGRPCHandler) SendInboxReply(ctx context.Context, req *pb.SendInboxReplyRequest) (*pb.EmailMessageRef, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	msg, apiErr := h.chatSvc.SendInboxReply(ctx, domain.SendInboxReplyInput{
		ConversationID: req.ConversationId,
		Subject:        req.Subject,
		Body:           req.Body,
		Cc:             req.Cc,
		AgentConfigID:  req.AgentConfigId,
		AgentRunID:     req.AgentRunId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.EmailMessageRef{MessageId: msg.ID, ConversationId: msg.ConversationID}, nil
}

func (h *emailBridgeGRPCHandler) PostReplyDraft(ctx context.Context, req *pb.PostReplyDraftRequest) (*pb.EmailMessageRef, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	msg, apiErr := h.chatSvc.PostReplyDraft(ctx, domain.PostReplyDraftInput{
		ConversationID:        req.ConversationId,
		Body:                  req.Body,
		Subject:               req.Subject,
		AgentConfigID:         req.AgentConfigId,
		AgentRunID:            req.AgentRunId,
		SourceThreadMessageID: req.SourceThreadMessageId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.EmailMessageRef{MessageId: msg.ID, ConversationId: msg.ConversationID}, nil
}

func (h *emailBridgeGRPCHandler) CreateEmailDomain(ctx context.Context, req *pb.CreateEmailDomainRequest) (*pb.EmailDomainInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	dom, apiErr := h.emailBridgeSvc.CreateDomain(ctx, req.Domain)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return emailDomainToProto(dom), nil
}

func (h *emailBridgeGRPCHandler) ListEmailDomains(ctx context.Context, req *pb.ListEmailDomainsRequest) (*pb.ListEmailDomainsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	domains, apiErr := h.emailBridgeSvc.ListDomains(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.EmailDomainInfo, 0, len(domains))
	for _, d := range domains {
		out = append(out, emailDomainToProto(d))
	}
	return &pb.ListEmailDomainsResponse{Domains: out}, nil
}

func (h *emailBridgeGRPCHandler) GetEmailDomain(ctx context.Context, req *pb.GetEmailDomainRequest) (*pb.EmailDomainInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	dom, apiErr := h.emailBridgeSvc.GetDomain(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return emailDomainToProto(dom), nil
}

func (h *emailBridgeGRPCHandler) VerifyEmailDomain(ctx context.Context, req *pb.VerifyEmailDomainRequest) (*pb.EmailDomainInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	dom, apiErr := h.emailBridgeSvc.VerifyDomain(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return emailDomainToProto(dom), nil
}

func (h *emailBridgeGRPCHandler) CreateEmailInbox(ctx context.Context, req *pb.CreateEmailInboxRequest) (*pb.EmailInboxInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	inbox, apiErr := h.emailBridgeSvc.CreateInbox(ctx, domain.CreateEmailInboxInput{
		EmailDomainID:        req.EmailDomainId,
		Address:              req.Address,
		FromName:             req.FromName,
		AgentConfigID:        req.AgentConfigId,
		AgentTriggerPolicy:   req.AgentTriggerPolicy,
		AgentTriggerKeywords: req.AgentTriggerKeywords,
		GroupID:              req.GroupId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return h.emailInboxToProto(inbox), nil
}

func (h *emailBridgeGRPCHandler) ListEmailInboxes(ctx context.Context, req *pb.ListEmailInboxesRequest) (*pb.ListEmailInboxesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	inboxes, apiErr := h.emailBridgeSvc.ListInboxes(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	out := make([]*pb.EmailInboxInfo, 0, len(inboxes))
	for _, i := range inboxes {
		out = append(out, h.emailInboxToProto(i))
	}
	return &pb.ListEmailInboxesResponse{Inboxes: out}, nil
}

func (h *emailBridgeGRPCHandler) GetEmailInbox(ctx context.Context, req *pb.GetEmailInboxRequest) (*pb.EmailInboxInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	inbox, apiErr := h.emailBridgeSvc.GetInbox(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return h.emailInboxToProto(inbox), nil
}

func (h *emailBridgeGRPCHandler) UpdateEmailInbox(ctx context.Context, req *pb.UpdateEmailInboxRequest) (*pb.EmailInboxInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	inbox, apiErr := h.emailBridgeSvc.UpdateInbox(ctx, req.Id, domain.UpdateEmailInboxInput{
		FromName:             req.FromName,
		Status:               req.Status,
		AgentConfigID:        req.AgentConfigId,
		AgentTriggerPolicy:   req.AgentTriggerPolicy,
		AgentTriggerKeywords: req.AgentTriggerKeywords,
		GroupID:              req.GroupId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return h.emailInboxToProto(inbox), nil
}

func (h *emailBridgeGRPCHandler) DeleteEmailInbox(ctx context.Context, req *pb.DeleteEmailInboxRequest) (*pb.EmailBridgeAck, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.emailBridgeSvc.DeleteInbox(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.EmailBridgeAck{Ok: true}, nil
}

func emailDomainToProto(d *domain.EmailDomain) *pb.EmailDomainInfo {
	return &pb.EmailDomainInfo{
		Id:         d.ID,
		AccountId:  d.AccountID,
		Domain:     d.Domain,
		Status:     d.Status,
		DkimTokens: d.DkimTokens,
		VerifiedAt: nullableTimestamp(d.VerifiedAt),
		CreatedAt:  timestamppb.New(d.CreatedAt),
		UpdatedAt:  timestamppb.New(d.UpdatedAt),
	}
}

func (h *emailBridgeGRPCHandler) emailInboxToProto(i *domain.EmailInbox) *pb.EmailInboxInfo {
	out := &pb.EmailInboxInfo{
		Id:                   i.ID,
		AccountId:            i.AccountID,
		EmailDomainId:        i.EmailDomainID,
		Address:              i.Address,
		FromName:             i.FromName,
		Status:               i.Status,
		AgentConfigId:        i.AgentConfigID,
		AgentTriggerPolicy:   i.AgentTriggerPolicy,
		AgentTriggerKeywords: i.AgentTriggerKeywords,
		GroupId:              i.GroupID,
		CreatedAt:            timestamppb.New(i.CreatedAt),
		UpdatedAt:            timestamppb.New(i.UpdatedAt),
	}
	// The forwarding address is derived, not stored: the inbox id is the local part on the Augno receiving
	// subdomain. Customers who can't repoint their apex MX forward their support address here; ingestion
	// resolves it back to this inbox by id (see resolveInbox). Omitted when the subdomain isn't configured.
	if h.inboundEmailDomain != "" {
		addr := i.ID + "@" + h.inboundEmailDomain
		out.ForwardingAddress = &addr
	}
	return out
}

func nullableTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
