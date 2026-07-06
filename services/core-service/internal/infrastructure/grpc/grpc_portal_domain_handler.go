package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type portalDomainGRPCHandler struct {
	pb.UnimplementedCorePortalDomainServiceServer

	portalDomainSvc domain.PortalDomainSvc
}

func RegisterPortalDomainService(server *grpc.Server, portalDomainSvc domain.PortalDomainSvc) {
	pb.RegisterCorePortalDomainServiceServer(server, &portalDomainGRPCHandler{portalDomainSvc: portalDomainSvc})
}

func portalDomainToProto(pd *domain.PortalDomain) *pb.PortalDomainInfo {
	if pd == nil {
		return nil
	}

	records := make([]*pb.PortalDNSRecord, 0, len(pd.DNSRecords))
	for _, r := range pd.DNSRecords {
		records = append(records, &pb.PortalDNSRecord{
			Type:   string(r.Type),
			Name:   r.Name,
			Value:  r.Value,
			Reason: string(r.Reason),
		})
	}

	info := &pb.PortalDomainInfo{
		Id:         pd.ID,
		AccountId:  pd.AccountID,
		Domain:     pd.Domain,
		Status:     string(pd.Status),
		DnsRecords: records,
		CreatedAt:  timestamppb.New(pd.CreatedAt),
		UpdatedAt:  timestamppb.New(pd.UpdatedAt),
	}
	if pd.VerifiedAt != nil {
		info.VerifiedAt = timestamppb.New(*pd.VerifiedAt)
	}

	return info
}

func (h *portalDomainGRPCHandler) CreatePortalDomain(ctx context.Context, req *pb.CreatePortalDomainRequest) (*pb.CreatePortalDomainResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	pd, apiErr := h.portalDomainSvc.CreatePortalDomain(ctx, req.Domain)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreatePortalDomainResponse{PortalDomain: portalDomainToProto(pd)}, nil
}

func (h *portalDomainGRPCHandler) GetPortalDomain(ctx context.Context, req *pb.GetPortalDomainRequest) (*pb.GetPortalDomainResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	pd, apiErr := h.portalDomainSvc.GetPortalDomain(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPortalDomainResponse{PortalDomain: portalDomainToProto(pd)}, nil
}

func (h *portalDomainGRPCHandler) ListPortalDomains(ctx context.Context, req *pb.ListPortalDomainsRequest) (*pb.ListPortalDomainsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	domains, apiErr := h.portalDomainSvc.ListPortalDomains(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.PortalDomainInfo, 0, len(domains))
	for _, pd := range domains {
		out = append(out, portalDomainToProto(pd))
	}
	return &pb.ListPortalDomainsResponse{PortalDomains: out}, nil
}

func (h *portalDomainGRPCHandler) VerifyPortalDomain(ctx context.Context, req *pb.VerifyPortalDomainRequest) (*pb.VerifyPortalDomainResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	pd, apiErr := h.portalDomainSvc.VerifyPortalDomain(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VerifyPortalDomainResponse{PortalDomain: portalDomainToProto(pd)}, nil
}

func (h *portalDomainGRPCHandler) DeletePortalDomain(ctx context.Context, req *pb.DeletePortalDomainRequest) (*pb.DeletePortalDomainResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.portalDomainSvc.DeletePortalDomain(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeletePortalDomainResponse{}, nil
}

func (h *portalDomainGRPCHandler) ResolvePortalHost(ctx context.Context, req *pb.ResolvePortalHostRequest) (*pb.ResolvePortalHostResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	account, apiErr := h.portalDomainSvc.ResolvePortalHost(ctx, req.Domain)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ResolvePortalHostResponse{Account: publicAccountToProto(account)}, nil
}

func (h *portalDomainGRPCHandler) BatchGetPortalDomainsByIDs(ctx context.Context, req *pb.BatchGetPortalDomainsByIDsRequest) (*pb.BatchGetPortalDomainsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	domains, apiErr := h.portalDomainSvc.BatchGetPortalDomainsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.PortalDomainInfo, 0, len(domains))
	for _, pd := range domains {
		out = append(out, portalDomainToProto(pd))
	}
	return &pb.BatchGetPortalDomainsByIDsResponse{PortalDomains: out}, nil
}
