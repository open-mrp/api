package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListAllocationEntries(ctx context.Context, req *pb.ListAllocationEntriesRequest) (*pb.ListAllocationEntriesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAllocationEntriesParams{
		Cursor:          req.Cursor,
		Limit:           req.Limit,
		Query:           req.Query,
		TransactionType: req.TransactionType,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.transactionAllocationSvc.ListAllocationEntries(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	entries := make([]*pb.AllocationEntryInfo, len(result.Entries))
	for i, e := range result.Entries {
		entries[i] = allocationEntryToProto(e)
	}

	return &pb.ListAllocationEntriesResponse{
		Entries: entries,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) UpdateTransactionAllocation(ctx context.Context, req *pb.UpdateTransactionAllocationRequest) (*pb.UpdateTransactionAllocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateTransactionAllocationParams{
		AllocationID: req.Id,
		Amount:       req.Amount,
	}

	allocation, apiErr := h.transactionAllocationSvc.UpdateTransactionAllocation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateTransactionAllocationResponse{
		Allocation: transactionAllocationToProto(allocation),
	}, nil
}

func (h *gRPCHandler) DeleteTransactionAllocation(ctx context.Context, req *pb.DeleteTransactionAllocationRequest) (*pb.DeleteTransactionAllocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.transactionAllocationSvc.DeleteTransactionAllocation(ctx, domain.DeleteTransactionAllocationParams{
		AllocationID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteTransactionAllocationResponse{}, nil
}

func (h *gRPCHandler) ListOpenCredits(ctx context.Context, req *pb.ListOpenCreditsRequest) (*pb.ListOpenCreditsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListOpenCreditsParams{
		CustomerIDs: req.CustomerIds,
		Cursor:      req.Cursor,
		Limit:       req.Limit,
	}

	if req.Query != nil {
		params.SearchQuery = req.Query
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.transactionAllocationSvc.ListOpenCredits(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEntries := make([]*pb.OpenCreditEntryInfo, len(result.Entries))
	for i, e := range result.Entries {
		pbEntries[i] = openCreditEntryToProto(e)
	}

	return &pb.ListOpenCreditsResponse{
		Entries: pbEntries,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func allocationEntryToProto(e *domain.AllocationEntry) *pb.AllocationEntryInfo {
	if e == nil {
		return nil
	}

	return &pb.AllocationEntryInfo{
		Id:                e.ID,
		AmountValue:       e.AmountValue,
		AmountUnitAbbr:    e.AmountUnitAbbr,
		CustomerName:      e.CustomerName,
		CustomerNumber:    e.CustomerNumber,
		TransactionId:     e.TransactionID,
		TransactionType:   e.TransactionType,
		TransactionMethod: e.TransactionMethod,
		AdjustmentType:    e.AdjustmentType,
		InvoiceId:         e.InvoiceID,
		InvoiceNumber:     e.InvoiceNumber,
		Note:              e.Note,
		CreatedAt:         timestamppb.New(e.CreatedAt),
	}
}

func openCreditEntryToProto(e *domain.OpenCreditEntry) *pb.OpenCreditEntryInfo {
	if e == nil {
		return nil
	}

	allocations := make([]*pb.OpenCreditInvoiceAllocationInfo, len(e.InvoiceAllocations))
	for i, a := range e.InvoiceAllocations {
		allocations[i] = &pb.OpenCreditInvoiceAllocationInfo{
			InvoiceNumber: a.InvoiceNumber,
			Amount:        a.Amount,
		}
	}

	return &pb.OpenCreditEntryInfo{
		Id:                  e.ID,
		Number:              e.Number,
		OriginalAmount:      e.OriginalAmount,
		AllocatedAmount:     e.AllocatedAmount,
		LeftoverAmount:      e.LeftoverAmount,
		CustomerId:          e.CustomerID,
		CustomerName:        e.CustomerName,
		CustomerNumber:      e.CustomerNumber,
		TransactionType:     e.TransactionType,
		TransactionMethod:   e.TransactionMethod,
		AdjustmentType:      e.AdjustmentType,
		ResponsibleUserName: e.ResponsibleUserName,
		Note:                e.Note,
		StripePaymentId:     e.StripePaymentID,
		InvoiceAllocations:  allocations,
		CreatedAt:           timestamppb.New(e.CreatedAt),
	}
}
