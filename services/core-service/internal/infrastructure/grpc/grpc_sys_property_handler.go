package grpc

import (
	"context"
	"strconv"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sysPropertyToProto(sp *domain.SysProperty) *pb.SysPropertyInfo {
	if sp == nil {
		return nil
	}
	return &pb.SysPropertyInfo{
		Id:        sp.ID,
		TypeId:    sp.TypeID,
		TypeCode:  string(sp.TypeCode),
		TypeName:  sp.TypeName,
		Value:     sp.Value,
		AccountId: sp.AccountID,
		CreatedAt: timestamppb.New(sp.CreatedAt),
		UpdatedAt: timestamppb.New(sp.UpdatedAt),
	}
}

func (h *gRPCHandler) ListSysProperties(ctx context.Context, req *pb.ListSysPropertiesRequest) (*pb.ListSysPropertiesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSysPropertiesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.sysPropertySvc.ListSysProperties(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.SysPropertyInfo, len(result.SysProperties))
	for i, sp := range result.SysProperties {
		pbItems[i] = sysPropertyToProto(sp)
	}

	return &pb.ListSysPropertiesResponse{
		SysProperties: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetSysProperty(ctx context.Context, req *pb.GetSysPropertyRequest) (*pb.GetSysPropertyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sp, apiErr := h.sysPropertySvc.GetSysProperty(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSysPropertyResponse{
		SysProperty: sysPropertyToProto(sp),
	}, nil
}

func (h *gRPCHandler) BatchGetSysPropertiesByIDs(ctx context.Context, req *pb.BatchGetSysPropertiesByIDsRequest) (*pb.BatchGetSysPropertiesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	items, apiErr := h.sysPropertySvc.BatchGetSysPropertiesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbItems := make([]*pb.SysPropertyInfo, len(items))
	for i, sp := range items {
		pbItems[i] = sysPropertyToProto(sp)
	}
	return &pb.BatchGetSysPropertiesByIDsResponse{SysProperties: pbItems}, nil
}

func (h *gRPCHandler) UpdateSysProperty(ctx context.Context, req *pb.UpdateSysPropertyRequest) (*pb.UpdateSysPropertyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSysPropertyParams{
		ID:    req.Id,
		Value: req.Value,
	}

	sp, apiErr := h.sysPropertySvc.UpdateSysProperty(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateSysPropertyResponse{
		SysProperty: sysPropertyToProto(sp),
	}, nil
}

func (h *gRPCHandler) GetLatestSysPropertyValue(ctx context.Context, req *pb.GetLatestSysPropertyValueRequest) (*pb.GetLatestSysPropertyValueResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	typeCode := constants.SysPropertyTypeCode(req.TypeCode)

	value, apiErr := h.sysPropertySvc.GetLatestSysPropertyValue(ctx, typeCode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	intValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInternalError(err, "Failed to parse sys property value as integer."))
	}

	return &pb.GetLatestSysPropertyValueResponse{
		Value: int32(intValue),
	}, nil
}
