package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegisterTerritoryService(server *grpc.Server, territorySvc domain.TerritorySvc) {
	handler.territorySvc = territorySvc
}

func territoryToProto(t *domain.Territory) *pb.TerritoryInfo {
	if t == nil {
		return nil
	}
	info := &pb.TerritoryInfo{
		Id:        t.ID,
		State:     t.State,
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
	}

	if t.StartZipcode != nil {
		info.StartZipcode = t.StartZipcode
	}
	if t.EndZipcode != nil {
		info.EndZipcode = t.EndZipcode
	}

	if t.SalesRep != nil {
		sr := &pb.TerritoryAccountUserInfo{
			Id:    t.SalesRep.ID,
			Name:  t.SalesRep.Name,
			Email: t.SalesRep.Email,
		}
		if t.SalesRep.Status != nil {
			s := string(*t.SalesRep.Status)
			sr.Status = &s
		}
		if t.SalesRep.CreatedAt != nil {
			sr.CreatedAt = timestamppb.New(*t.SalesRep.CreatedAt)
		}
		if t.SalesRep.UpdatedAt != nil {
			sr.UpdatedAt = timestamppb.New(*t.SalesRep.UpdatedAt)
		}
		info.SalesRep = sr
	}

	if t.ProductLine != nil {
		pl := &pb.TerritoryProductLineInfo{
			Id:   t.ProductLine.ID,
			Name: t.ProductLine.Name,
		}
		if t.ProductLine.CommissionPolicy != nil {
			isCommissionExempt := t.ProductLine.CommissionPolicy.ToBool()
			pl.IsCommissionExempt = &isCommissionExempt
		}
		if t.ProductLine.FreightPolicy != nil {
			isFreightExempt := t.ProductLine.FreightPolicy.ToBool()
			pl.IsFreightExempt = &isFreightExempt
		}
		if t.ProductLine.CreatedAt != nil {
			pl.CreatedAt = timestamppb.New(*t.ProductLine.CreatedAt)
		}
		if t.ProductLine.UpdatedAt != nil {
			pl.UpdatedAt = timestamppb.New(*t.ProductLine.UpdatedAt)
		}
		info.ProductLine = pl
	}

	return info
}

func (h *gRPCHandler) ListTerritories(ctx context.Context, req *pb.ListTerritoriesRequest) (*pb.ListTerritoriesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.territorySvc.ListTerritories(ctx, domain.ListTerritoriesParams{
		AccountID: req.AccountId,
		Cursor:    req.Cursor,
		Limit:     req.Limit,
		Query:     req.Query,
		Includes:  req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	territories := make([]*pb.TerritoryInfo, len(result.Territories))
	for i, t := range result.Territories {
		territories[i] = territoryToProto(t)
	}

	return &pb.ListTerritoriesResponse{
		Territories: territories,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetTerritory(ctx context.Context, req *pb.GetTerritoryRequest) (*pb.GetTerritoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	territory, apiErr := h.territorySvc.GetTerritory(ctx, domain.GetTerritoryParams{
		AccountID:   req.AccountId,
		TerritoryID: req.Id,
		Includes:    req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetTerritoryResponse{
		Territory: territoryToProto(territory),
	}, nil
}

func (h *gRPCHandler) CreateTerritory(ctx context.Context, req *pb.CreateTerritoryRequest) (*pb.CreateTerritoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	territory, apiErr := h.territorySvc.CreateTerritory(ctx, domain.CreateTerritoryParams{
		AccountID:     req.AccountId,
		State:         req.State,
		StartZipcode:  req.StartZipcode,
		EndZipcode:    req.EndZipcode,
		SalesRepID:    req.SalesRepId,
		ProductLineID: req.ProductLineId,
		Includes:      req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateTerritoryResponse{
		Territory: territoryToProto(territory),
	}, nil
}

func (h *gRPCHandler) UpdateTerritory(ctx context.Context, req *pb.UpdateTerritoryRequest) (*pb.UpdateTerritoryResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	territory, apiErr := h.territorySvc.UpdateTerritory(ctx, domain.UpdateTerritoryParams{
		AccountID:         req.AccountId,
		TerritoryID:       req.Id,
		State:             req.State,
		StartZipcode:      req.StartZipcode,
		EndZipcode:        req.EndZipcode,
		SalesRepID:        req.SalesRepId,
		ProductLineID:     req.ProductLineId,
		ClearProductLine:  req.ClearProductLine,
		ClearStartZipcode: req.ClearStartZipcode,
		ClearEndZipcode:   req.ClearEndZipcode,
		Includes:          req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateTerritoryResponse{
		Territory: territoryToProto(territory),
	}, nil
}

func (h *gRPCHandler) DeleteTerritory(ctx context.Context, req *pb.DeleteTerritoryRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.territorySvc.DeleteTerritory(ctx, domain.DeleteTerritoryParams{
		AccountID:   req.AccountId,
		TerritoryID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
