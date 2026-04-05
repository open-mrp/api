package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListTransactionsParams{
		Cursor:              req.Cursor,
		Limit:               req.Limit,
		Query:               req.Query,
		Status:              req.Status,
		TypeCodes:           req.TypeCodes,
		AdjustmentTypeCodes: req.AdjustmentTypeCodes,
		MethodCodes:         req.MethodCodes,
		CustomerIDs:         req.CustomerIds,
		CustomerGroupIDs:    req.CustomerGroupIds,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.transactionSvc.ListTransactions(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	transactions := make([]*pb.TransactionSummaryInfo, len(result.Transactions))
	for i, t := range result.Transactions {
		transactions[i] = transactionSummaryToProto(t)
	}

	return &pb.ListTransactionsResponse{
		Transactions: transactions,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	transaction, apiErr := h.transactionSvc.GetTransaction(ctx, domain.GetTransactionParams{
		TransactionID: req.Id,
		Includes:      req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *gRPCHandler) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateTransactionParams{
		CustomerID:          req.CustomerId,
		TransactionTypeCode: req.TransactionTypeCode,
		Amount:              req.Amount,
	}
	if req.TransactionMethodCode != nil {
		params.TransactionMethodCode = req.TransactionMethodCode
	}
	if req.AdjustmentTypeCode != nil {
		params.AdjustmentTypeCode = req.AdjustmentTypeCode
	}
	if req.ResponsibleUserId != nil {
		params.ResponsibleUserID = req.ResponsibleUserId
	}
	if req.Note != nil {
		params.Note = req.Note
	}
	if req.StripePaymentId != nil {
		params.StripePaymentID = req.StripePaymentId
	}

	transaction, apiErr := h.transactionSvc.CreateTransaction(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *gRPCHandler) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.UpdateTransactionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateTransactionParams{
		TransactionID:          req.Id,
		ClearResponsibleUser:   req.ClearResponsibleUser,
		ClearTransactionMethod: req.ClearTransactionMethod,
		ClearAdjustmentType:    req.ClearAdjustmentType,
	}
	if req.Number != nil {
		params.Number = req.Number
	}
	if req.Note != nil {
		params.Note = req.Note
	}
	if req.Amount != nil {
		params.Amount = req.Amount
	}
	if req.TransactionMethodCode != nil {
		params.TransactionMethodCode = req.TransactionMethodCode
	}
	if req.AdjustmentTypeCode != nil {
		params.AdjustmentTypeCode = req.AdjustmentTypeCode
	}
	if req.ResponsibleUserId != nil {
		params.ResponsibleUserID = req.ResponsibleUserId
	}
	if req.IsFullyAllocated != nil {
		params.IsFullyAllocated = req.IsFullyAllocated
	}
	transaction, apiErr := h.transactionSvc.UpdateTransaction(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *gRPCHandler) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	transaction, apiErr := h.transactionSvc.DeleteTransaction(ctx, domain.DeleteTransactionParams{
		TransactionID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteTransactionResponse{
		Transaction: transactionToProto(transaction),
	}, nil
}

func (h *gRPCHandler) ListAccountTransactions(ctx context.Context, req *pb.ListAccountTransactionsRequest) (*pb.ListAccountTransactionsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAccountTransactionsParams{
		CustomerAccountID:    req.CustomerAccountId,
		Cursor:               req.Cursor,
		Limit:                req.Limit,
		Query:                req.Query,
		Status:               req.Status,
		Type:                 req.Type,
		IncludeChildAccounts: req.IncludeChildAccounts,
	}

	result, apiErr := h.transactionSvc.ListAccountTransactions(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	transactions := make([]*pb.TransactionInfo, len(result.Transactions))
	for i, t := range result.Transactions {
		transactions[i] = transactionToProto(t)
	}

	return &pb.ListAccountTransactionsResponse{
		Transactions: transactions,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func transactionToProto(t *domain.Transaction) *pb.TransactionInfo {
	if t == nil {
		return nil
	}

	info := &pb.TransactionInfo{
		Id:                     t.ID,
		Number:                 t.Number,
		AmountId:               t.AmountID,
		AmountValue:            t.AmountValue,
		AmountUnitId:           t.AmountUnitID,
		AmountUnitAbbreviation: t.AmountUnitAbbr,
		CustomerId:             t.CustomerID,
		CustomerName:           t.CustomerName,
		CustomerNumber:         t.CustomerNumber,
		ResponsibleUserId:      t.ResponsibleUserID,
		ResponsibleUserName:    t.ResponsibleUserName,
		Note:                   t.Note,
		TransactionTypeCode:    t.TransactionTypeCode,
		TransactionTypeName:    t.TransactionTypeName,
		TransactionTypeId:      t.TransactionTypeID,
		TransactionMethodCode:  t.TransactionMethodCode,
		TransactionMethodName:  t.TransactionMethodName,
		TransactionMethodId:    t.TransactionMethodID,
		AdjustmentTypeCode:     t.AdjustmentTypeCode,
		AdjustmentTypeName:     t.AdjustmentTypeName,
		AdjustmentTypeId:       t.AdjustmentTypeID,
		IsFullyAllocated:       t.IsFullyAllocated,
		StripePaymentId:        t.StripePaymentID,
		AllocationCount:        t.AllocationCount,
		CreatedAt:              timestamppb.New(t.CreatedAt),
		UpdatedAt:              timestamppb.New(t.UpdatedAt),
	}

	if t.Allocations != nil {
		allocations := make([]*pb.TransactionAllocationInfo, len(t.Allocations))
		for i, a := range t.Allocations {
			allocations[i] = transactionAllocationToProto(a)
		}
		info.Allocations = allocations
	}

	return info
}

func transactionSummaryToProto(t *domain.TransactionSummary) *pb.TransactionSummaryInfo {
	if t == nil {
		return nil
	}

	return &pb.TransactionSummaryInfo{
		Id:                     t.ID,
		Number:                 t.Number,
		AmountId:               t.AmountID,
		AmountValue:            t.AmountValue,
		AmountUnitId:           t.AmountUnitID,
		AmountUnitAbbreviation: t.AmountUnitAbbr,
		CustomerId:             t.CustomerID,
		CustomerName:           t.CustomerName,
		CustomerNumber:         t.CustomerNumber,
		TransactionTypeCode:    t.TransactionTypeCode,
		TransactionTypeName:    t.TransactionTypeName,
		TransactionTypeId:      t.TransactionTypeID,
		TransactionMethodCode:  t.TransactionMethodCode,
		TransactionMethodName:  t.TransactionMethodName,
		TransactionMethodId:    t.TransactionMethodID,
		AdjustmentTypeCode:     t.AdjustmentTypeCode,
		AdjustmentTypeName:     t.AdjustmentTypeName,
		AdjustmentTypeId:       t.AdjustmentTypeID,
		IsFullyAllocated:       t.IsFullyAllocated,
		AllocationCount:        t.AllocationCount,
		CreatedAt:              timestamppb.New(t.CreatedAt),
		UpdatedAt:              timestamppb.New(t.UpdatedAt),
	}
}
