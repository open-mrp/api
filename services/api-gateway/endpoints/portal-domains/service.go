// Package portaldomainep implements the customer portal custom-domain management endpoints, backed by the core-service CorePortalDomainService gRPC client.
package portaldomainep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

// PortalDomainSvc backs the portal domain endpoints. The account is derived from the caller identity propagated over gRPC metadata, so requests never embed it.
type PortalDomainSvc interface {
	CreatePortalDomain(ctx context.Context, req *CreatePortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError)
	ListPortalDomains(ctx context.Context, req *ListPortalDomainsRequest) (*apiresource.List[apiresource.PortalDomain], *apierror.APIError)
	GetPortalDomain(ctx context.Context, req *GetPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError)
	VerifyPortalDomain(ctx context.Context, req *VerifyPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError)
	DeletePortalDomain(ctx context.Context, req *DeletePortalDomainRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ResolvePortalHost(ctx context.Context, req *ResolvePortalHostRequest) (*apiresource.PublicAccount, *apierror.APIError)
}

type PortalDomainSvcConfig struct {
	// PortalDomainClient (required) is the core-service CorePortalDomainService gRPC client.
	PortalDomainClient pb.CorePortalDomainServiceClient
}

type portalDomainSvcImpl struct {
	portalDomainClient pb.CorePortalDomainServiceClient
}

var portalDomainSvcTracer = tracing.GetTracer("api-gateway.endpoints.portal-domains.service")

func (c *PortalDomainSvcConfig) validate() error {
	if c.PortalDomainClient == nil {
		return fmt.Errorf("portal domain endpoint service: portal domain client is required")
	}
	return nil
}

func NewPortalDomainSvc(config *PortalDomainSvcConfig) PortalDomainSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &portalDomainSvcImpl{portalDomainClient: config.PortalDomainClient}
}

func portalDomainFromProto(d *pb.PortalDomainInfo) *apiresource.PortalDomain {
	records := make([]apiresource.DNSRecord, 0, len(d.DnsRecords))
	for _, r := range d.DnsRecords {
		records = append(records, apiresource.DNSRecord{
			Object: constants.ObjectTypeDNSRecord,
			Type:   constants.DNSRecordType(r.Type),
			Name:   r.Name,
			Value:  r.Value,
			Reason: constants.DNSRecordReason(r.Reason),
		})
	}

	return &apiresource.PortalDomain{
		ID:         d.Id,
		Object:     constants.ObjectTypePortalDomain,
		Domain:     d.Domain,
		Status:     constants.PortalDomainStatus(d.Status),
		DNSRecords: apiresource.NewList(records, apiresource.PageInfo{}),
		VerifiedAt: grpcutil.TimestampToTimePtr(d.VerifiedAt),
		CreatedAt:  grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(d.UpdatedAt),
	}
}

func publicAccountFromProto(a *pb.PublicAccountInfo) *apiresource.PublicAccount {
	return &apiresource.PublicAccount{
		ID:           a.Id,
		Object:       constants.ObjectTypePublicAccount,
		Name:         a.Name,
		Slug:         a.Slug,
		SupportEmail: a.SupportEmail,
		LogoURL:      a.LogoUrl,
	}
}

func (s *portalDomainSvcImpl) CreatePortalDomain(ctx context.Context, req *CreatePortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
	pbReq := &pb.CreatePortalDomainRequest{Domain: req.Domain}
	resp, apiErr := grpcutil.CallRPC(ctx, portalDomainSvcTracer, "service.portal_domains.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePortalDomainResponse, error) {
			return s.portalDomainClient.CreatePortalDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalDomainFromProto(resp.PortalDomain), nil
}

func (s *portalDomainSvcImpl) ListPortalDomains(ctx context.Context, _ *ListPortalDomainsRequest) (*apiresource.List[apiresource.PortalDomain], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, portalDomainSvcTracer, "service.portal_domains.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPortalDomainsResponse, error) {
			return s.portalDomainClient.ListPortalDomains(ctx, &pb.ListPortalDomainsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.PortalDomain, len(resp.PortalDomains))
	for i, d := range resp.PortalDomains {
		items[i] = *portalDomainFromProto(d)
	}
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (s *portalDomainSvcImpl) GetPortalDomain(ctx context.Context, req *GetPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
	pbReq := &pb.GetPortalDomainRequest{Id: req.ID}
	resp, apiErr := grpcutil.CallRPC(ctx, portalDomainSvcTracer, "service.portal_domains.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPortalDomainResponse, error) {
			return s.portalDomainClient.GetPortalDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalDomainFromProto(resp.PortalDomain), nil
}

func (s *portalDomainSvcImpl) VerifyPortalDomain(ctx context.Context, req *VerifyPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
	pbReq := &pb.VerifyPortalDomainRequest{Id: req.ID}
	resp, apiErr := grpcutil.CallRPC(ctx, portalDomainSvcTracer, "service.portal_domains.verify", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VerifyPortalDomainResponse, error) {
			return s.portalDomainClient.VerifyPortalDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return portalDomainFromProto(resp.PortalDomain), nil
}

func (s *portalDomainSvcImpl) DeletePortalDomain(ctx context.Context, req *DeletePortalDomainRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeletePortalDomainRequest{Id: req.ID}
	_, apiErr := grpcutil.CallRPC(ctx, portalDomainSvcTracer, "service.portal_domains.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeletePortalDomainResponse, error) {
			return s.portalDomainClient.DeletePortalDomain(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (s *portalDomainSvcImpl) ResolvePortalHost(ctx context.Context, req *ResolvePortalHostRequest) (*apiresource.PublicAccount, *apierror.APIError) {
	pbReq := &pb.ResolvePortalHostRequest{Domain: req.Domain}
	resp, apiErr := grpcutil.CallRPC(ctx, portalDomainSvcTracer, "service.portal_domains.resolve_host", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ResolvePortalHostResponse, error) {
			return s.portalDomainClient.ResolvePortalHost(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return publicAccountFromProto(resp.Account), nil
}
