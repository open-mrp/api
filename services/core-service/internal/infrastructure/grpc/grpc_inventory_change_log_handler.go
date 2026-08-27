package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func inventoryChangeLogToProto(icl *domain.InventoryChangeLog) *pb.InventoryChangeLogInfo {
	if icl == nil {
		return nil
	}

	info := &pb.InventoryChangeLogInfo{
		Id:                            icl.ID,
		ItemId:                        icl.ItemID,
		ItemSku:                       icl.ItemSKU,
		ItemCreatedAt:                 timestamppb.New(icl.ItemCreatedAt),
		ItemUpdatedAt:                 timestamppb.New(icl.ItemUpdatedAt),
		QuantityId:                    icl.QuantityID,
		QuantityValue:                 icl.QuantityValue,
		QuantityUnitId:                icl.QuantityUnitID,
		QuantityUnitName:              icl.QuantityUnitName,
		QuantityUnitAbbreviation:      icl.QuantityUnitAbbreviation,
		QuantityUnitType:              icl.QuantityUnitType,
		QuantityUnitRatioNumerator:    icl.QuantityUnitRatioNumerator,
		QuantityUnitRatioDenominator:  icl.QuantityUnitRatioDenominator,
		QuantityUnitOffsetNumerator:   icl.QuantityUnitOffsetNumerator,
		QuantityUnitOffsetDenominator: icl.QuantityUnitOffsetDenominator,
		QuantityUnitCreatedAt:         timestamppb.New(icl.QuantityUnitCreatedAt),
		QuantityUnitUpdatedAt:         timestamppb.New(icl.QuantityUnitUpdatedAt),
		ActionTypeCode:                icl.ActionTypeCode,
		ScanningStationId:             icl.ScanningStationID,
		ScanningStationName:           icl.ScanningStationName,
		ResponsibleUserId:             icl.ResponsibleUserID,
		ResponsibleUserName:           icl.ResponsibleUserName,
		CreatedAt:                     timestamppb.New(icl.CreatedAt),
		UpdatedAt:                     timestamppb.New(icl.UpdatedAt),
	}

	if icl.ItemTypeCode != nil {
		info.ItemTypeCode = icl.ItemTypeCode
	}
	if icl.ScanningStationType != nil {
		info.ScanningStationType = icl.ScanningStationType
	}
	if icl.ScanningStationCreatedAt != nil {
		info.ScanningStationCreatedAt = timestamppb.New(*icl.ScanningStationCreatedAt)
	}
	if icl.ScanningStationUpdatedAt != nil {
		info.ScanningStationUpdatedAt = timestamppb.New(*icl.ScanningStationUpdatedAt)
	}
	if icl.ResponsibleUserCreatedAt != nil {
		info.ResponsibleUserCreatedAt = timestamppb.New(*icl.ResponsibleUserCreatedAt)
	}
	if icl.ResponsibleUserUpdatedAt != nil {
		info.ResponsibleUserUpdatedAt = timestamppb.New(*icl.ResponsibleUserUpdatedAt)
	}

	return info
}

func (h *gRPCHandler) ListInventoryChangeLogs(ctx context.Context, req *pb.ListInventoryChangeLogsRequest) (*pb.ListInventoryChangeLogsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var startDate, endDate *timestamppb.Timestamp
	startDate = req.StartDate
	endDate = req.EndDate

	params := domain.ListInventoryChangeLogsParams{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		ItemIDs:          req.ItemIds,
		ActionTypeCodes:  req.ActionTypeCodes,
		ChangedByUserIDs: req.ChangedByUserIds,
	}

	if startDate != nil {
		t := startDate.AsTime()
		params.StartDate = &t
	}
	if endDate != nil {
		t := endDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.inventoryChangeLogSvc.ListInventoryChangeLogs(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.InventoryChangeLogInfo, len(result.Items))
	for i, icl := range result.Items {
		pbItems[i] = inventoryChangeLogToProto(icl)
	}

	return &pb.ListInventoryChangeLogsResponse{
		InventoryChangeLogs: pbItems,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetInventoryChangeLog(ctx context.Context, req *pb.GetInventoryChangeLogRequest) (*pb.GetInventoryChangeLogResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	icl, apiErr := h.inventoryChangeLogSvc.GetInventoryChangeLog(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetInventoryChangeLogResponse{
		InventoryChangeLog: inventoryChangeLogToProto(icl),
	}, nil
}

func (h *gRPCHandler) ExportInventoryChangeLogs(ctx context.Context, req *pb.ExportInventoryChangeLogsRequest) (*pb.ExportInventoryChangeLogsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ExportInventoryChangeLogsParams{
		ItemIDs:          req.ItemIds,
		ActionTypeCodes:  req.ActionTypeCodes,
		ChangedByUserIDs: req.ChangedByUserIds,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	items, apiErr := h.inventoryChangeLogSvc.ExportInventoryChangeLogs(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.InventoryChangeLogInfo, len(items))
	for i, icl := range items {
		pbItems[i] = inventoryChangeLogToProto(icl)
	}

	return &pb.ExportInventoryChangeLogsResponse{
		InventoryChangeLogs: pbItems,
		Count:               int64(len(items)),
	}, nil
}
