package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func registrationFlowOptionToProto(opt *domain.RegistrationFlowOption) *pb.RegistrationFlowOptionInfo {
	return &pb.RegistrationFlowOptionInfo{
		Id:   opt.ID,
		Name: opt.Name,
	}
}

func registrationFlowToProto(rf *domain.RegistrationFlow) *pb.RegistrationFlowInfo {
	cgOptions := make([]*pb.RegistrationFlowOptionInfo, len(rf.CustomerGroupOptions))
	for i, opt := range rf.CustomerGroupOptions {
		cgOptions[i] = registrationFlowOptionToProto(opt)
	}

	ptOptions := make([]*pb.RegistrationFlowOptionInfo, len(rf.PaymentTermOptions))
	for i, opt := range rf.PaymentTermOptions {
		ptOptions[i] = registrationFlowOptionToProto(opt)
	}

	stOptions := make([]*pb.RegistrationFlowOptionInfo, len(rf.ShippingTermOptions))
	for i, opt := range rf.ShippingTermOptions {
		stOptions[i] = registrationFlowOptionToProto(opt)
	}

	return &pb.RegistrationFlowInfo{
		Id:                   rf.ID,
		Name:                 rf.Name,
		CustomerGroupOptions: cgOptions,
		PaymentTermOptions:   ptOptions,
		ShippingTermOptions:  stOptions,
		CreatedAt:            timestamppb.New(rf.CreatedAt),
		UpdatedAt:            timestamppb.New(rf.UpdatedAt),
	}
}

func (h *gRPCHandler) ListRegistrationFlows(ctx context.Context, req *pb.ListRegistrationFlowsRequest) (*pb.ListRegistrationFlowsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListRegistrationFlowsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.registrationFlowSvc.ListRegistrationFlows(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbFlows := make([]*pb.RegistrationFlowInfo, len(result.RegistrationFlows))
	for i, rf := range result.RegistrationFlows {
		pbFlows[i] = registrationFlowToProto(rf)
	}

	return &pb.ListRegistrationFlowsResponse{
		RegistrationFlows: pbFlows,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetRegistrationFlow(ctx context.Context, req *pb.GetRegistrationFlowRequest) (*pb.GetRegistrationFlowResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	flow, apiErr := h.registrationFlowSvc.GetRegistrationFlow(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRegistrationFlowResponse{
		RegistrationFlow: registrationFlowToProto(flow),
	}, nil
}

func (h *gRPCHandler) CreateRegistrationFlow(ctx context.Context, req *pb.CreateRegistrationFlowRequest) (*pb.CreateRegistrationFlowResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateRegistrationFlowParams{
		Name:             req.Name,
		CustomerGroupIDs: req.CustomerGroupIds,
		PaymentTermIDs:   req.PaymentTermIds,
		ShippingTermIDs:  req.ShippingTermIds,
	}

	flow, apiErr := h.registrationFlowSvc.CreateRegistrationFlow(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateRegistrationFlowResponse{
		RegistrationFlow: registrationFlowToProto(flow),
	}, nil
}

func (h *gRPCHandler) UpdateRegistrationFlow(ctx context.Context, req *pb.UpdateRegistrationFlowRequest) (*pb.UpdateRegistrationFlowResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateRegistrationFlowParams{
		RegistrationFlowID:  req.Id,
		Name:                req.Name,
		CustomerGroupIDs:    req.CustomerGroupIds,
		PaymentTermIDs:      req.PaymentTermIds,
		ShippingTermIDs:     req.ShippingTermIds,
		HasCustomerGroupIDs: req.HasCustomerGroupIds,
		HasPaymentTermIDs:   req.HasPaymentTermIds,
		HasShippingTermIDs:  req.HasShippingTermIds,
	}

	flow, apiErr := h.registrationFlowSvc.UpdateRegistrationFlow(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateRegistrationFlowResponse{
		RegistrationFlow: registrationFlowToProto(flow),
	}, nil
}

func (h *gRPCHandler) DeleteRegistrationFlow(ctx context.Context, req *pb.DeleteRegistrationFlowRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.registrationFlowSvc.DeleteRegistrationFlow(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) GetRegistrationFlowBySlug(ctx context.Context, req *pb.GetRegistrationFlowBySlugRequest) (*pb.GetRegistrationFlowBySlugResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	flow, apiErr := h.registrationFlowSvc.GetRegistrationFlowBySlug(ctx, req.Slug)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRegistrationFlowBySlugResponse{
		RegistrationFlow: registrationFlowToProto(flow),
	}, nil
}

func (h *gRPCHandler) RegisterCustomer(ctx context.Context, req *pb.RegisterCustomerRequest) (*pb.RegisterCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.RegisterCustomerParams{
		AccountSlug:        req.AccountSlug,
		IsExistingCustomer: req.IsExistingCustomer,
		CustomerData: domain.CustomerRegistrationData{
			Number:          req.CustomerNumber,
			Name:            req.CustomerName,
			CustomerGroupID: req.CustomerGroupId,
			Phone:           req.Phone,
			ShippingTermID:  req.ShippingTermId,
			PaymentTermID:   req.PaymentTermId,
		},
	}

	if req.Address != nil {
		params.CustomerData.Address = &domain.CustomerRegistrationAddress{
			StreetLine1: req.Address.StreetLine_1,
			StreetLine2: req.Address.StreetLine_2,
			Locality:    req.Address.Locality,
			State:       req.Address.State,
			PostalCode:  req.Address.PostalCode,
			Country:     req.Address.Country,
			Name:        req.Address.Name,
		}
	}

	apiErr := h.registrationFlowSvc.RegisterCustomer(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RegisterCustomerResponse{}, nil
}
