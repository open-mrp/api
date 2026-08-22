package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
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

func (h *gRPCHandler) BatchGetPropertiesByIDs(ctx context.Context, req *pb.BatchGetPropertiesByIDsRequest) (*pb.BatchGetPropertiesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	properties, apiErr := h.propertySvc.BatchGetPropertiesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbProps := make([]*pb.PropertyInfo, len(properties))
	for i, p := range properties {
		pbProps[i] = propertyToProto(p)
	}
	return &pb.BatchGetPropertiesByIDsResponse{Properties: pbProps}, nil
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

func (h *gRPCHandler) BulkUpsertProperties(ctx context.Context, req *pb.BulkUpsertPropertiesRequest) (*pb.BulkUpsertPropertiesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	inputs := make([]domain.UpsertPropertyParams, len(req.Properties))
	for i, p := range req.Properties {
		attributes := make([]domain.UpsertPropertyAttributeParams, len(p.Attributes))
		for j, a := range p.Attributes {
			attributes[j] = domain.UpsertPropertyAttributeParams{
				Value:     a.Value,
				ColorCode: a.ColorCode,
			}
		}
		inputs[i] = domain.UpsertPropertyParams{
			Name:       p.Name,
			Attributes: attributes,
		}
	}

	job, apiErr := h.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: inputs,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.BulkUpsertPropertiesResponse{Job: jobToProto(job)}, nil
}

func (h *gRPCHandler) ExportProperties(ctx context.Context, req *pb.ExportPropertiesRequest) (*pb.ExportPropertiesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	job, apiErr := h.propertySvc.ExportProperties(ctx, domain.ExportPropertiesParams{Query: req.Query})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ExportPropertiesResponse{Job: jobToProto(job)}, nil
}
