package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func attributeToProto(a *domain.Attribute) *pb.AttributeInfo {
	if a == nil {
		return nil
	}
	return &pb.AttributeInfo{
		Id:        a.ID,
		Value:     a.Value,
		ColorCode: a.ColorCode,
		SortOrder: a.SortOrder,
		IsPublic:  a.IsPublic,
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
}

func (h *gRPCHandler) ListAttributes(ctx context.Context, req *pb.ListAttributesRequest) (*pb.ListAttributesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAttributesParams{
		PropertyID: req.PropertyId,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
	}

	result, apiErr := h.attributeSvc.ListAttributes(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbAttributes := make([]*pb.AttributeInfo, len(result.Attributes))
	for i, a := range result.Attributes {
		pbAttributes[i] = attributeToProto(a)
	}

	return &pb.ListAttributesResponse{
		Attributes: pbAttributes,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAttribute(ctx context.Context, req *pb.GetAttributeRequest) (*pb.GetAttributeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	attribute, apiErr := h.attributeSvc.GetAttribute(ctx, req.PropertyId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAttributeResponse{
		Attribute: attributeToProto(attribute),
	}, nil
}

func (h *gRPCHandler) CreateAttribute(ctx context.Context, req *pb.CreateAttributeRequest) (*pb.CreateAttributeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateAttributeParams{
		PropertyID: req.PropertyId,
		Value:      req.Value,
		ColorCode:  req.ColorCode,
		SortOrder:  req.SortOrder,
	}

	attribute, apiErr := h.attributeSvc.CreateAttribute(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAttributeResponse{
		Attribute: attributeToProto(attribute),
	}, nil
}

func (h *gRPCHandler) UpdateAttribute(ctx context.Context, req *pb.UpdateAttributeRequest) (*pb.UpdateAttributeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateAttributeParams{
		AttributeID: req.Id,
		PropertyID:  req.PropertyId,
		Value:       req.Value,
		ColorCode:   req.ColorCode,
		SortOrder:   req.SortOrder,
	}

	attribute, apiErr := h.attributeSvc.UpdateAttribute(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAttributeResponse{
		Attribute: attributeToProto(attribute),
	}, nil
}

func (h *gRPCHandler) BatchGetAttributesByIDs(ctx context.Context, req *pb.BatchGetAttributesByIDsRequest) (*pb.BatchGetAttributesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	attributes, apiErr := h.attributeSvc.BatchGetAttributesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbAttrs := make([]*pb.AttributeInfo, len(attributes))
	for i, a := range attributes {
		pbAttrs[i] = attributeToProto(a)
	}
	return &pb.BatchGetAttributesByIDsResponse{Attributes: pbAttrs}, nil
}

func (h *gRPCHandler) DeleteAttribute(ctx context.Context, req *pb.DeleteAttributeRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.attributeSvc.DeleteAttribute(ctx, domain.DeleteAttributeParams{
		AttributeID: req.Id,
		PropertyID:  req.PropertyId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
