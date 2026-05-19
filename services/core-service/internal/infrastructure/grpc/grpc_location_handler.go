package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func locationChildToProto(c domain.LocationChild) *pb.LocationChildInfo {
	return &pb.LocationChildInfo{
		Id:       c.ID,
		Name:     c.Name,
		TypeCode: c.TypeCode,
	}
}

func locationToProto(sl *domain.Location) *pb.LocationInfo {
	if sl == nil {
		return nil
	}

	info := &pb.LocationInfo{
		Id:             sl.ID,
		Name:           sl.Name,
		TypeCode:       sl.TypeCode,
		ParentId:       sl.ParentID,
		ParentName:     sl.ParentName,
		ParentTypeCode: sl.ParentTypeCode,
		CreatedAt:      timestamppb.New(sl.CreatedAt),
		UpdatedAt:      timestamppb.New(sl.UpdatedAt),
	}

	if sl.Children != nil {
		info.Children = make([]*pb.LocationChildInfo, len(sl.Children))
		for i, c := range sl.Children {
			info.Children[i] = locationChildToProto(c)
		}
	}

	return info
}

func locationTypeToProto(slt *domain.LocationType) *pb.LocationTypeInfo {
	if slt == nil {
		return nil
	}
	return &pb.LocationTypeInfo{
		Id:        slt.ID,
		Code:      slt.Code,
		Name:      slt.Name,
		CreatedAt: timestamppb.New(slt.CreatedAt),
		UpdatedAt: timestamppb.New(slt.UpdatedAt),
	}
}

func (h *gRPCHandler) ListLocations(ctx context.Context, req *pb.ListLocationsRequest) (*pb.ListLocationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListLocationsParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: req.Includes,
	}

	result, apiErr := h.locationSvc.ListLocations(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbLocations := make([]*pb.LocationInfo, len(result.Locations))
	for i, sl := range result.Locations {
		pbLocations[i] = locationToProto(sl)
	}

	return &pb.ListLocationsResponse{
		Locations: pbLocations,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetLocation(ctx context.Context, req *pb.GetLocationRequest) (*pb.GetLocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sl, apiErr := h.locationSvc.GetLocation(ctx, domain.GetLocationParams{
		LocationID: req.Id,
		Includes:   req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetLocationResponse{
		Location: locationToProto(sl),
	}, nil
}

func (h *gRPCHandler) CreateLocation(ctx context.Context, req *pb.CreateLocationRequest) (*pb.CreateLocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateLocationParams{
		Name:     req.Name,
		TypeCode: req.TypeCode,
		ParentID: req.ParentId,
		ChildIDs: req.ChildIds,
		Includes: req.Includes,
	}

	sl, apiErr := h.locationSvc.CreateLocation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateLocationResponse{
		Location: locationToProto(sl),
	}, nil
}

func (h *gRPCHandler) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.UpdateLocationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateLocationParams{
		LocationID: req.Id,
		Name:       req.Name,
		TypeCode:   req.TypeCode,
		ParentID:   patch.StringFieldFromProto(req.ParentId),
		ChildIDs:   stringListPatchToSliceField(req.ChildIds),
		Includes:   req.Includes,
	}

	sl, apiErr := h.locationSvc.UpdateLocation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateLocationResponse{
		Location: locationToProto(sl),
	}, nil
}

func (h *gRPCHandler) DeleteLocation(ctx context.Context, req *pb.DeleteLocationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{
		LocationID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) GetLocationType(ctx context.Context, req *pb.GetLocationTypeRequest) (*pb.GetLocationTypeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	lt, apiErr := h.locationSvc.GetLocationType(ctx, domain.GetLocationTypeParams{
		Identifier: req.Identifier,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetLocationTypeResponse{
		LocationType: locationTypeToProto(lt),
	}, nil
}

func (h *gRPCHandler) ListLocationTypes(ctx context.Context, req *pb.ListLocationTypesRequest) (*pb.ListLocationTypesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListLocationTypesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.locationSvc.ListLocationTypes(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbTypes := make([]*pb.LocationTypeInfo, len(result.LocationTypes))
	for i, slt := range result.LocationTypes {
		pbTypes[i] = locationTypeToProto(slt)
	}

	return &pb.ListLocationTypesResponse{
		LocationTypes: pbTypes,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}
