package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListSettlements(ctx context.Context, req *pb.ListSettlementsRequest) (*pb.ListSettlementsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSettlementsParams{
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		Query:          req.Query,
		TransactionIDs: req.TransactionIds,
		InvoiceIDs:     req.InvoiceIds,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.settlementSvc.ListSettlements(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	settlements := make([]*pb.SettlementSummaryInfo, len(result.Settlements))
	for i, s := range result.Settlements {
		settlements[i] = settlementSummaryToProto(s)
	}

	return &pb.ListSettlementsResponse{
		Settlements: settlements,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetSettlement(ctx context.Context, req *pb.GetSettlementRequest) (*pb.GetSettlementResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	settlement, apiErr := h.settlementSvc.GetSettlement(ctx, domain.GetSettlementParams{
		SettlementID: req.Id,
		Includes:     req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSettlementResponse{
		Settlement: settlementToProto(settlement),
	}, nil
}

func (h *gRPCHandler) CreateSettlement(ctx context.Context, req *pb.CreateSettlementRequest) (*pb.CreateSettlementResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	allocations := make([]domain.CreateSettlementAllocationParams, len(req.Allocations))
	for i, a := range req.Allocations {
		allocations[i] = domain.CreateSettlementAllocationParams{
			TransactionID: a.TransactionId,
			InvoiceID:     a.InvoiceId,
			Amount:        a.Amount,
			Note:          a.Note,
		}
	}

	settlement, apiErr := h.settlementSvc.CreateSettlement(ctx, domain.CreateSettlementParams{
		ResponsibleUserID: req.ResponsibleUserId,
		Allocations:       allocations,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSettlementResponse{
		Settlement: settlementToProto(settlement),
	}, nil
}

func (h *gRPCHandler) UpdateSettlement(ctx context.Context, req *pb.UpdateSettlementRequest) (*pb.UpdateSettlementResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSettlementParams{
		SettlementID: req.Id,
	}
	if req.Number != nil {
		params.Number = req.Number
	}
	if req.Note != nil {
		params.Note = req.Note
	}
	if req.ResponsibleUserId != nil {
		params.ResponsibleUserID = req.ResponsibleUserId
	}

	settlement, apiErr := h.settlementSvc.UpdateSettlement(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateSettlementResponse{
		Settlement: settlementToProto(settlement),
	}, nil
}

func (h *gRPCHandler) DeleteSettlement(ctx context.Context, req *pb.DeleteSettlementRequest) (*pb.DeleteSettlementResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	settlement, apiErr := h.settlementSvc.DeleteSettlement(ctx, domain.DeleteSettlementParams{
		SettlementID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteSettlementResponse{
		Settlement: settlementToProto(settlement),
	}, nil
}

func settlementToProto(s *domain.Settlement) *pb.SettlementInfo {
	if s == nil {
		return nil
	}

	info := &pb.SettlementInfo{
		Id:                  s.ID,
		Number:              s.Number,
		Note:                s.Note,
		ResponsibleUserId:   s.ResponsibleUserID,
		ResponsibleUserName: s.ResponsibleUserName,
		CreatedAt:           timestamppb.New(s.CreatedAt),
		UpdatedAt:           timestamppb.New(s.UpdatedAt),
	}

	if s.Allocations != nil {
		allocations := make([]*pb.TransactionAllocationInfo, len(s.Allocations))
		for i, a := range s.Allocations {
			allocations[i] = transactionAllocationToProto(a)
		}
		info.Allocations = allocations
	}

	return info
}

func settlementSummaryToProto(s *domain.SettlementSummary) *pb.SettlementSummaryInfo {
	if s == nil {
		return nil
	}

	return &pb.SettlementSummaryInfo{
		Id:               s.ID,
		Number:           s.Number,
		AllocationCount:  s.AllocationCount,
		TotalPayments:    s.TotalPayments,
		TotalRebates:     s.TotalRebates,
		TotalAdjustments: s.TotalAdjustments,
		TotalCredits:     s.TotalCredits,
		InvoiceNumbers:   s.InvoiceNumbers,
		CustomerNames:    s.CustomerNames,
		CreatedAt:        timestamppb.New(s.CreatedAt),
		UpdatedAt:        timestamppb.New(s.UpdatedAt),
	}
}

func transactionAllocationToProto(a *domain.TransactionAllocation) *pb.TransactionAllocationInfo {
	if a == nil {
		return nil
	}

	info := &pb.TransactionAllocationInfo{
		Id:                     a.ID,
		AmountId:               a.AmountID,
		AmountValue:            a.AmountValue,
		AmountUnitId:           a.AmountUnitID,
		AmountUnitAbbreviation: a.AmountUnitAbbr,
		Note:                   a.Note,
		TransactionId:          a.TransactionID,
		CreatedAt:              timestamppb.New(a.CreatedAt),
		UpdatedAt:              timestamppb.New(a.UpdatedAt),
	}
	if a.InvoiceID != "" {
		info.InvoiceId = &a.InvoiceID
	}
	if a.InvoiceNumber != "" {
		info.InvoiceNumber = &a.InvoiceNumber
	}
	if a.TransactionNumber != "" {
		info.TransactionNumber = &a.TransactionNumber
	}
	if a.TransactionType != "" {
		info.TransactionType = &a.TransactionType
	}
	return info
}
