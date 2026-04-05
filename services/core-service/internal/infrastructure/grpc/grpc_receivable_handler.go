package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func receivableEntryToProto(e domain.ReceivableEntry) *pb.ReceivableEntryProto {
	p := &pb.ReceivableEntryProto{
		InvoiceId:        e.InvoiceID,
		InvoiceNumber:    e.InvoiceNumber,
		InvoicedAt:       timestamppb.New(e.InvoicedAt),
		CustomerId:       e.CustomerID,
		CustomerNumber:   e.CustomerNumber,
		CustomerName:     e.CustomerName,
		RemainingBalance: e.RemainingBalance,
		IsPaidInFull:     e.IsPaidInFull,
	}
	if e.PONumber != nil {
		p.PoNumber = e.PONumber
	}
	return p
}

func (h *gRPCHandler) ListReceivables(ctx context.Context, req *pb.ListReceivablesRequest) (*pb.ListReceivablesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListReceivablesParams{
		Limit: req.Limit,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.CutoffDate != nil {
		t := req.CutoffDate.AsTime()
		params.CutoffDate = &t
	}
	if req.Query != nil {
		params.Query = req.Query
	}

	result, apiErr := h.receivableSvc.ListReceivables(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	receivables := make([]*pb.ReceivableEntryProto, len(result.Items))
	for i, e := range result.Items {
		receivables[i] = receivableEntryToProto(e)
	}

	pageInfo := &pb.PageInfo{
		HasNextPage: result.PageString != nil,
	}
	if result.PageString != nil {
		pageInfo.NextCursor = result.PageString
	}

	return &pb.ListReceivablesResponse{
		Receivables: receivables,
		PageInfo:    pageInfo,
	}, nil
}

func (h *gRPCHandler) ListReceivablesByCustomer(ctx context.Context, req *pb.ListReceivablesByCustomerRequest) (*pb.ListReceivablesByCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListReceivablesByCustomerParams{
		CustomerAccountID: req.CustomerAccountId,
		Limit:             req.Limit,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.CutoffDate != nil {
		t := req.CutoffDate.AsTime()
		params.CutoffDate = &t
	}
	if req.Query != nil {
		params.Query = req.Query
	}

	result, apiErr := h.receivableSvc.ListReceivablesByCustomer(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	receivables := make([]*pb.ReceivableEntryProto, len(result.Items))
	for i, e := range result.Items {
		receivables[i] = receivableEntryToProto(e)
	}

	pageInfo := &pb.PageInfo{
		HasNextPage: result.PageString != nil,
	}
	if result.PageString != nil {
		pageInfo.NextCursor = result.PageString
	}

	return &pb.ListReceivablesByCustomerResponse{
		Receivables: receivables,
		PageInfo:    pageInfo,
	}, nil
}

func (h *gRPCHandler) ExportReceivablesByCustomer(ctx context.Context, req *pb.ExportReceivablesByCustomerRequest) (*pb.ExportReceivablesByCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListReceivablesByCustomerParams{
		CustomerAccountID: req.CustomerAccountId,
	}

	if req.CutoffDate != nil {
		t := req.CutoffDate.AsTime()
		params.CutoffDate = &t
	}

	entries, apiErr := h.receivableSvc.ExportReceivablesByCustomer(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	receivables := make([]*pb.ReceivableEntryProto, len(entries))
	for i, e := range entries {
		receivables[i] = receivableEntryToProto(e)
	}

	return &pb.ExportReceivablesByCustomerResponse{
		Receivables: receivables,
	}, nil
}

func (h *gRPCHandler) EmailReceivablesForCustomer(ctx context.Context, req *pb.EmailReceivablesForCustomerRequest) (*pb.EmailReceivablesForCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.EmailReceivablesParams{
		CustomerAccountID: req.CustomerAccountId,
		RecipientEmails:   req.RecipientEmails,
	}

	apiErr := h.receivableSvc.EmailReceivablesForCustomer(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.EmailReceivablesForCustomerResponse{}, nil
}
