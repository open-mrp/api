package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func propertyToProto(p *domain.Property) *pb.PropertyInfo {
	if p == nil {
		return nil
	}

	info := &pb.PropertyInfo{
		Id:        p.ID,
		Name:      p.Name,
		IsPublic:  p.IsPublic,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}

	if len(p.Attributes) > 0 {
		info.Attributes = make([]*pb.AttributeInfo, len(p.Attributes))
		for i, a := range p.Attributes {
			info.Attributes[i] = attributeToProto(a)
		}
	}

	return info
}

func (h *gRPCHandler) ListProperties(ctx context.Context, req *pb.ListPropertiesRequest) (*pb.ListPropertiesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPropertiesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.propertySvc.ListProperties(ctx, params, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProperties := make([]*pb.PropertyInfo, len(result.Properties))
	for i, p := range result.Properties {
		pbProperties[i] = propertyToProto(p)
	}

	return &pb.ListPropertiesResponse{
		Properties: pbProperties,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetProperty(ctx context.Context, req *pb.GetPropertyRequest) (*pb.GetPropertyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	property, apiErr := h.propertySvc.GetProperty(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPropertyResponse{
		Property: propertyToProto(property),
	}, nil
}

func (h *gRPCHandler) CreateProperty(ctx context.Context, req *pb.CreatePropertyRequest) (*pb.CreatePropertyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreatePropertyParams{
		Name: req.Name,
	}

	property, apiErr := h.propertySvc.CreateProperty(ctx, params, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreatePropertyResponse{
		Property: propertyToProto(property),
	}, nil
}

func (h *gRPCHandler) UpdateProperty(ctx context.Context, req *pb.UpdatePropertyRequest) (*pb.UpdatePropertyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePropertyParams{
		PropertyID: req.Id,
		Name:       req.Name,
	}

	property, apiErr := h.propertySvc.UpdateProperty(ctx, params, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePropertyResponse{
		Property: propertyToProto(property),
	}, nil
}

func (h *gRPCHandler) DeleteProperty(ctx context.Context, req *pb.DeletePropertyRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.propertySvc.DeleteProperty(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
