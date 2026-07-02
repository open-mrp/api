package emailbridgeep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

// EmailBridgeSvc backs the email-domain and email-inbox management endpoints via the
// notification-service EmailBridgeService gRPC client. The account is derived from the caller
// identity propagated over gRPC metadata, so requests never embed it.
type EmailBridgeSvc interface {
	CreateDomain(ctx context.Context, req *CreateEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError)
	ListDomains(ctx context.Context, req *ListEmailDomainsRequest) (*apiresource.List[apiresource.EmailDomain], *apierror.APIError)
	GetDomain(ctx context.Context, req *GetEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError)
	VerifyDomain(ctx context.Context, req *VerifyEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError)

	CreateInbox(ctx context.Context, req *CreateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError)
	ListInboxes(ctx context.Context, req *ListEmailInboxesRequest) (*apiresource.List[apiresource.EmailInbox], *apierror.APIError)
	GetInbox(ctx context.Context, req *GetEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError)
	UpdateInbox(ctx context.Context, req *UpdateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError)
	DeleteInbox(ctx context.Context, req *DeleteEmailInboxRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type EmailBridgeSvcConfig struct {
	// EmailBridgeClient (required) is the notification-service EmailBridgeService gRPC client.
	EmailBridgeClient pb.EmailBridgeServiceClient
}

type emailBridgeSvcImpl struct {
	emailBridgeClient pb.EmailBridgeServiceClient
}

var emailBridgeSvcTracer = tracing.GetTracer("api-gateway.endpoints.email-bridge.service")

func (c *EmailBridgeSvcConfig) validate() error {
	if c.EmailBridgeClient == nil {
		return fmt.Errorf("email bridge endpoint service: email bridge client is required")
	}
	return nil
}

func NewEmailBridgeSvc(config *EmailBridgeSvcConfig) EmailBridgeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &emailBridgeSvcImpl{emailBridgeClient: config.EmailBridgeClient}
}

func emailDomainFromProto(d *pb.EmailDomainInfo) *apiresource.EmailDomain {
	return &apiresource.EmailDomain{
		ID:         d.Id,
		Object:     constants.ObjectTypeEmailDomain,
		Domain:     d.Domain,
		Status:     d.Status,
		DkimTokens: emptyIfNil(d.DkimTokens),
		VerifiedAt: grpcutil.TimestampToTimePtr(d.VerifiedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

// emailInboxFromProto maps a proto inbox to a clean resource. The expandable email_domain and
// agent_config fields are left nil (per the include-gating convention); their FK ids are stashed in
// the request-scoped LoadMeta so the resolver hydrates them only when ?include= requests them.
func emailInboxFromProto(ctx context.Context, i *pb.EmailInboxInfo) *apiresource.EmailInbox {
	inbox := &apiresource.EmailInbox{
		ID:                   i.Id,
		Object:               constants.ObjectTypeEmailInbox,
		Address:              i.Address,
		FromName:             i.FromName,
		Status:               i.Status,
		AgentTriggerPolicy:   i.AgentTriggerPolicy,
		AgentTriggerKeywords: emptyIfNil(i.AgentTriggerKeywords),
		CreatedAt:            grpcutil.TimestampToTime(i.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(i.UpdatedAt),
	}
	meta := resourcekit.GetLoadMeta(ctx)
	if i.EmailDomainId != "" {
		meta.Set(constants.ObjectTypeEmailInbox, inbox.ID, "email_domain_id", i.EmailDomainId)
	}
	if i.AgentConfigId != nil && *i.AgentConfigId != "" {
		meta.Set(constants.ObjectTypeEmailInbox, inbox.ID, "agent_config_id", *i.AgentConfigId)
	}
	return inbox
}

// emptyIfNil normalizes a nil slice to a non-nil empty slice so required array
// fields serialize as `[]` rather than `null` (dkim_tokens / agent_trigger_keywords
// are stored NULL when empty, but the API contract types them as non-nullable arrays).
func emptyIfNil(vals []string) []string {
	if vals == nil {
		return []string{}
	}
	return vals
}

func (s *emailBridgeSvcImpl) CreateDomain(ctx context.Context, req *CreateEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
	pbReq := &pb.CreateEmailDomainRequest{Domain: req.Domain}
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.create_domain", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailDomainInfo, error) {
			return s.emailBridgeClient.CreateEmailDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return emailDomainFromProto(resp), nil
}

func (s *emailBridgeSvcImpl) ListDomains(ctx context.Context, _ *ListEmailDomainsRequest) (*apiresource.List[apiresource.EmailDomain], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.list_domains", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListEmailDomainsResponse, error) {
			return s.emailBridgeClient.ListEmailDomains(ctx, &pb.ListEmailDomainsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.EmailDomain, len(resp.Domains))
	for i, d := range resp.Domains {
		items[i] = *emailDomainFromProto(d)
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (s *emailBridgeSvcImpl) GetDomain(ctx context.Context, req *GetEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
	pbReq := &pb.GetEmailDomainRequest{Id: req.ID}
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.get_domain", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailDomainInfo, error) {
			return s.emailBridgeClient.GetEmailDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return emailDomainFromProto(resp), nil
}

func (s *emailBridgeSvcImpl) VerifyDomain(ctx context.Context, req *VerifyEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
	pbReq := &pb.VerifyEmailDomainRequest{Id: req.ID}
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.verify_domain", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailDomainInfo, error) {
			return s.emailBridgeClient.VerifyEmailDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return emailDomainFromProto(resp), nil
}

func (s *emailBridgeSvcImpl) CreateInbox(ctx context.Context, req *CreateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
	pbReq := &pb.CreateEmailInboxRequest{
		EmailDomainId:        req.EmailDomainID,
		Address:              req.Address,
		FromName:             req.FromName.Ptr(),
		AgentConfigId:        req.AgentConfigID.Ptr(),
		AgentTriggerPolicy:   req.AgentTriggerPolicy.Ptr(),
		AgentTriggerKeywords: req.AgentTriggerKeywords,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.create_inbox", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailInboxInfo, error) {
			return s.emailBridgeClient.CreateEmailInbox(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return emailInboxFromProto(ctx, resp), nil
}

func (s *emailBridgeSvcImpl) ListInboxes(ctx context.Context, _ *ListEmailInboxesRequest) (*apiresource.List[apiresource.EmailInbox], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.list_inboxes", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListEmailInboxesResponse, error) {
			return s.emailBridgeClient.ListEmailInboxes(ctx, &pb.ListEmailInboxesRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.EmailInbox, len(resp.Inboxes))
	for i, in := range resp.Inboxes {
		items[i] = *emailInboxFromProto(ctx, in)
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (s *emailBridgeSvcImpl) GetInbox(ctx context.Context, req *GetEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
	pbReq := &pb.GetEmailInboxRequest{Id: req.ID}
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.get_inbox", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailInboxInfo, error) {
			return s.emailBridgeClient.GetEmailInbox(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return emailInboxFromProto(ctx, resp), nil
}

func (s *emailBridgeSvcImpl) UpdateInbox(ctx context.Context, req *UpdateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
	pbReq := &pb.UpdateEmailInboxRequest{
		Id:                   req.ID,
		FromName:             req.FromName.Ptr(),
		Status:               req.Status,
		AgentConfigId:        req.AgentConfigID.Ptr(),
		AgentTriggerPolicy:   req.AgentTriggerPolicy.Ptr(),
		AgentTriggerKeywords: req.AgentTriggerKeywords,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.update_inbox", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailInboxInfo, error) {
			return s.emailBridgeClient.UpdateEmailInbox(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return emailInboxFromProto(ctx, resp), nil
}

func (s *emailBridgeSvcImpl) DeleteInbox(ctx context.Context, req *DeleteEmailInboxRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteEmailInboxRequest{Id: req.ID}
	_, apiErr := grpcutil.CallRPC(ctx, emailBridgeSvcTracer, "service.email_bridge.delete_inbox", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailBridgeAck, error) {
			return s.emailBridgeClient.DeleteEmailInbox(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}
