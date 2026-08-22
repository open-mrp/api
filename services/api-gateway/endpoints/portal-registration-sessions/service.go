// Package portalregsessionep implements the buyer customer-portal registration-session endpoints, backed by the core-service CoreService gRPC client.
package portalregsessionep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

// PortalRegistrationSessionSvc backs the buyer registration-session endpoints. The buyer is derived from the caller identity propagated over gRPC metadata.
type PortalRegistrationSessionSvc interface {
	CreateOrResume(ctx context.Context, req *CreateOrResumePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError)
	Get(ctx context.Context, req *GetPortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError)
	Update(ctx context.Context, req *UpdatePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError)
	Complete(ctx context.Context, req *CompletePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError)
	Abandon(ctx context.Context, req *AbandonPortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError)
	List(ctx context.Context, req *ListPortalRegistrationSessionsRequest) (*apiresource.List[apiresource.PortalRegistrationSession], *apierror.APIError)
}

type PortalRegistrationSessionSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

func (c *PortalRegistrationSessionSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("portal registration session endpoint service: core client is required")
	}
	return nil
}

type portalRegSessionSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var portalRegSessionSvcTracer = tracing.GetTracer("api-gateway.endpoints.portal-registration-sessions.service")

func NewPortalRegistrationSessionSvc(config *PortalRegistrationSessionSvcConfig) PortalRegistrationSessionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &portalRegSessionSvcImpl{coreClient: config.CoreClient}
}

func (s *portalRegSessionSvcImpl) CreateOrResume(ctx context.Context, req *CreateOrResumePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalRegSessionSvcTracer, "service.portal_registration_sessions.create_or_resume", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PortalRegistrationSessionResponse, error) {
			return s.coreClient.CreateOrResumePortalRegistrationSession(ctx, &pb.CreateOrResumePortalRegistrationSessionRequest{SellerSlug: req.SellerSlug}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalRegistrationSessionFromProto(resp.Session), nil
}

func (s *portalRegSessionSvcImpl) Get(ctx context.Context, req *GetPortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalRegSessionSvcTracer, "service.portal_registration_sessions.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PortalRegistrationSessionResponse, error) {
			return s.coreClient.GetPortalRegistrationSession(ctx, &pb.GetPortalRegistrationSessionRequest{TypeId: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalRegistrationSessionFromProto(resp.Session), nil
}

func (s *portalRegSessionSvcImpl) Update(ctx context.Context, req *UpdatePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalRegSessionSvcTracer, "service.portal_registration_sessions.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PortalRegistrationSessionResponse, error) {
			return s.coreClient.UpdatePortalRegistrationSession(ctx, &pb.UpdatePortalRegistrationSessionRequest{
				TypeId:             req.ID,
				Step:               string(req.Step),
				SessionData:        portalRegistrationSessionDataToProto(req.SessionData),
				IsExistingCustomer: req.IsExistingCustomer.Ptr(),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalRegistrationSessionFromProto(resp.Session), nil
}

func (s *portalRegSessionSvcImpl) Complete(ctx context.Context, req *CompletePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalRegSessionSvcTracer, "service.portal_registration_sessions.complete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PortalRegistrationSessionResponse, error) {
			return s.coreClient.CompletePortalRegistrationSession(ctx, &pb.CompletePortalRegistrationSessionRequest{TypeId: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalRegistrationSessionFromProto(resp.Session), nil
}

func (s *portalRegSessionSvcImpl) Abandon(ctx context.Context, req *AbandonPortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalRegSessionSvcTracer, "service.portal_registration_sessions.abandon", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PortalRegistrationSessionResponse, error) {
			return s.coreClient.AbandonPortalRegistrationSession(ctx, &pb.AbandonPortalRegistrationSessionRequest{TypeId: req.ID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalRegistrationSessionFromProto(resp.Session), nil
}

func (s *portalRegSessionSvcImpl) List(ctx context.Context, req *ListPortalRegistrationSessionsRequest) (*apiresource.List[apiresource.PortalRegistrationSession], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalRegSessionSvcTracer, "service.portal_registration_sessions.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPortalRegistrationSessionsResponse, error) {
			return s.coreClient.ListPortalRegistrationSessions(ctx, &pb.ListPortalRegistrationSessionsRequest{
				Cursor: req.Cursor,
				Limit:  req.Limit,
				Status: req.Status,
				Search: req.Query,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalRegistrationSessionListFromProto(ctx, resp), nil
}

func portalRegistrationSessionListFromProto(ctx context.Context, resp *pb.ListPortalRegistrationSessionsResponse) *apiresource.List[apiresource.PortalRegistrationSession] {
	if resp == nil {
		return apiresource.NewList[apiresource.PortalRegistrationSession](nil, apiresource.PageInfo{})
	}
	sessions := make([]apiresource.PortalRegistrationSession, 0, len(resp.Sessions))
	for _, s := range resp.Sessions {
		if mapped := portalRegistrationSessionFromProto(s); mapped != nil {
			sessions = append(sessions, *mapped)
		}
	}
	return apiresource.NewList(sessions, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func portalRegistrationSessionDataToProto(d *PortalRegistrationSessionDataInput) *pb.PortalRegistrationSessionData {
	if d == nil {
		return &pb.PortalRegistrationSessionData{}
	}
	return &pb.PortalRegistrationSessionData{
		CustomerName:      d.CustomerName,
		CustomerNumber:    d.CustomerNumber,
		CustomerGroupId:   d.CustomerGroupID,
		PaymentTermId:     d.PaymentTermID,
		ShippingTermId:    d.ShippingTermID,
		Phone:             d.Phone,
		AddressName:       d.AddressName,
		AddressStreet_1:   d.AddressStreet1,
		AddressStreet_2:   d.AddressStreet2,
		AddressLocality:   d.AddressLocality,
		AddressState:      d.AddressState,
		AddressPostalCode: d.AddressPostalCode,
		AddressCountry:    d.AddressCountry,
	}
}

func portalRegistrationSessionFromProto(s *pb.PortalRegistrationSessionInfo) *apiresource.PortalRegistrationSession {
	if s == nil {
		return nil
	}
	out := &apiresource.PortalRegistrationSession{
		ID:                 s.Id,
		Object:             constants.ObjectTypePortalRegistrationSession,
		SellerAccountID:    s.SellerAccountId,
		SellerSlug:         s.SellerSlug,
		UserID:             s.UserId,
		IsExistingCustomer: s.IsExistingCustomer,
		Step:               constants.PortalRegistrationStep(s.Step),
		Status:             constants.PortalRegistrationStatus(s.Status),
		CustomerID:         s.CustomerId,
		CreatedAt:          s.CreatedAt.AsTime(),
		UpdatedAt:          s.UpdatedAt.AsTime(),
	}
	if s.SessionData != nil {
		out.SessionData = &apiresource.PortalRegistrationSessionData{
			Object:            constants.ObjectTypePortalRegistrationSessionData,
			CustomerName:      s.SessionData.CustomerName,
			CustomerNumber:    s.SessionData.CustomerNumber,
			CustomerGroupID:   s.SessionData.CustomerGroupId,
			PaymentTermID:     s.SessionData.PaymentTermId,
			ShippingTermID:    s.SessionData.ShippingTermId,
			Phone:             s.SessionData.Phone,
			AddressName:       s.SessionData.AddressName,
			AddressStreet1:    s.SessionData.AddressStreet_1,
			AddressStreet2:    s.SessionData.AddressStreet_2,
			AddressLocality:   s.SessionData.AddressLocality,
			AddressState:      s.SessionData.AddressState,
			AddressPostalCode: s.SessionData.AddressPostalCode,
			AddressCountry:    s.SessionData.AddressCountry,
		}
	}
	if s.CompletedAt != nil {
		t := s.CompletedAt.AsTime()
		out.CompletedAt = &t
	}
	if s.AbandonedAt != nil {
		t := s.AbandonedAt.AsTime()
		out.AbandonedAt = &t
	}
	return out
}
