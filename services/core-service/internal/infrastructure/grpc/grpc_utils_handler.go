package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
)

func (h *gRPCHandler) CheckDuplicate(ctx context.Context, req *pb.CheckDuplicateRequest) (*pb.CheckDuplicateResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var checkType domain.DuplicateCheckType
	switch req.Type {
	case pb.DuplicateCheckType_DUPLICATE_CHECK_TYPE_INVOICE_NUMBER:
		checkType = domain.DuplicateCheckTypeInvoiceNumber
	case pb.DuplicateCheckType_DUPLICATE_CHECK_TYPE_ORDER_NUMBER:
		checkType = domain.DuplicateCheckTypeOrderNumber
	case pb.DuplicateCheckType_DUPLICATE_CHECK_TYPE_CUSTOMER_PO_NUMBER:
		checkType = domain.DuplicateCheckTypeCustomerPO
	default:
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.CheckDuplicateParams{
		Type:         checkType,
		RecordNumber: req.RecordNumber,
	}
	if req.CustomerId != nil {
		params.CustomerID = req.CustomerId
	}

	result, apiErr := h.utilsSvc.CheckDuplicate(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.CheckDuplicateResponse{
		IsDuplicate: result.IsDuplicate,
	}
	if result.Message != nil {
		resp.Message = result.Message
	}

	return resp, nil
}

func (h *gRPCHandler) EmailRecord(ctx context.Context, req *pb.EmailRecordRequest) (*pb.EmailRecordResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var recordType domain.EmailRecordType
	switch req.Type {
	case pb.EmailRecordType_EMAIL_RECORD_TYPE_INVOICE:
		recordType = domain.EmailRecordTypeInvoice
	case pb.EmailRecordType_EMAIL_RECORD_TYPE_SALES_ORDER:
		recordType = domain.EmailRecordTypeSalesOrder
	case pb.EmailRecordType_EMAIL_RECORD_TYPE_PURCHASE_ORDER:
		recordType = domain.EmailRecordTypePurchaseOrder
	default:
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.EmailRecordParams{
		ID:   req.Id,
		Type: recordType,
	}

	apiErr := h.utilsSvc.EmailRecord(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.EmailRecordResponse{}, nil
}

func (h *gRPCHandler) RequestDemo(ctx context.Context, req *pb.RequestDemoRequest) (*pb.RequestDemoResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.RequestDemoParams{
		Name:    req.Name,
		Email:   req.Email,
		Company: req.Company,
	}
	if req.PhoneNumber != nil {
		params.PhoneNumber = req.PhoneNumber
	}
	if req.Message != nil {
		params.Message = req.Message
	}

	apiErr := h.utilsSvc.RequestDemo(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RequestDemoResponse{
		Message: "Demo request submitted successfully.",
	}, nil
}

func (h *gRPCHandler) SubmitFeedback(ctx context.Context, req *pb.SubmitFeedbackRequest) (*pb.SubmitFeedbackResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.SubmitFeedbackParams{
		Question: req.Question,
		Answer:   req.Answer,
	}
	if req.PageUrl != nil {
		params.PageURL = req.PageUrl
	}

	apiErr := h.utilsSvc.SubmitFeedback(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SubmitFeedbackResponse{
		Message: "Feedback submitted successfully.",
	}, nil
}
