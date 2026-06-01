package inventorychangelogep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/export"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InventoryChangeLogSvc interface {
	ListInventoryChangeLogs(ctx context.Context, req *ListInventoryChangeLogsRequest) (*apiresource.List[apiresource.InventoryChangeLog], *apierror.APIError)
	GetInventoryChangeLog(ctx context.Context, req *RetrieveInventoryChangeLogRequest) (*apiresource.InventoryChangeLog, *apierror.APIError)
	ExportInventoryChangeLogs(ctx context.Context, req *ExportInventoryChangeLogsRequest) (*httptransport.FileDownload, *apierror.APIError)
}

type InventoryChangeLogSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type inventoryChangeLogSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var inventoryChangeLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.inventory_change_logs.service")

func (c *InventoryChangeLogSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("inventory change log endpoint service: core client is required")
	}
	return nil
}

func NewInventoryChangeLogSvc(config *InventoryChangeLogSvcConfig) InventoryChangeLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &inventoryChangeLogSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *inventoryChangeLogSvcImpl) ListInventoryChangeLogs(ctx context.Context, req *ListInventoryChangeLogsRequest) (*apiresource.List[apiresource.InventoryChangeLog], *apierror.APIError) {
	pbReq := &pb.ListInventoryChangeLogsRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		ItemIds:          req.ItemIDs,
		ActionTypeCodes:  req.ActionTypeCodes,
		ChangedByUserIds: req.ChangedByUserIDs,
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, inventoryChangeLogSvcTracer, "service.inventory_change_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInventoryChangeLogsResponse, error) {
			return m.coreClient.ListInventoryChangeLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	items := make([]apiresource.InventoryChangeLog, len(resp.InventoryChangeLogs))
	for i, icl := range resp.InventoryChangeLogs {
		items[i] = inventoryChangeLogFromProto(icl)
		stashInventoryChangeLogMeta(meta, icl)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *inventoryChangeLogSvcImpl) GetInventoryChangeLog(ctx context.Context, req *RetrieveInventoryChangeLogRequest) (*apiresource.InventoryChangeLog, *apierror.APIError) {
	pbReq := &pb.GetInventoryChangeLogRequest{
		Id: req.InventoryChangeLogID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, inventoryChangeLogSvcTracer, "service.inventory_change_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetInventoryChangeLogResponse, error) {
			return m.coreClient.GetInventoryChangeLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := inventoryChangeLogFromProto(resp.InventoryChangeLog)
	stashInventoryChangeLogMeta(meta, resp.InventoryChangeLog)
	return &result, nil
}

func (m *inventoryChangeLogSvcImpl) ExportInventoryChangeLogs(ctx context.Context, req *ExportInventoryChangeLogsRequest) (*httptransport.FileDownload, *apierror.APIError) {
	pbReq := &pb.ExportInventoryChangeLogsRequest{
		ItemIds:          req.ItemIDs,
		ActionTypeCodes:  req.ActionTypeCodes,
		ChangedByUserIds: req.ChangedByUserIDs,
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, inventoryChangeLogSvcTracer, "service.inventory_change_logs.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportInventoryChangeLogsResponse, error) {
			return m.coreClient.ExportInventoryChangeLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	body, err := export.InventoryChangeLogsToExcel(resp)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	startDateStr := "all"
	if req.StartDate != nil {
		startDateStr = req.StartDate.Format("2006-01-02")
	}
	endDateStr := "all"
	if req.EndDate != nil {
		endDateStr = req.EndDate.Format("2006-01-02")
	}
	filename := fmt.Sprintf("inventory-change-logs-%s-%s.xlsx", startDateStr, endDateStr)

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    filename,
		Body:        body,
	}, nil
}

func inventoryChangeLogFromProto(icl *pb.InventoryChangeLogInfo) apiresource.InventoryChangeLog {
	if icl == nil {
		return apiresource.InventoryChangeLog{}
	}

	return apiresource.InventoryChangeLog{
		ID:             icl.Id,
		Object:         constants.ObjectTypeInventoryChangeLog,
		ActionTypeCode: constants.InventoryActionType(icl.ActionTypeCode),
		CreatedAt:      grpcutil.TimestampToTime(icl.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(icl.UpdatedAt),
	}
}

func stashInventoryChangeLogMeta(meta *resourcekit.LoadMeta, icl *pb.InventoryChangeLogInfo) {
	if icl == nil {
		return
	}

	item := &apiresource.Item{
		ID:        icl.ItemId,
		Object:    constants.ObjectTypeItem,
		SKU:       icl.ItemSku,
		CreatedAt: grpcutil.TimestampToTime(icl.ItemCreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(icl.ItemUpdatedAt),
	}
	if icl.ItemTypeCode != nil {
		item.ItemTypeCode = constants.ItemTypeCode(*icl.ItemTypeCode)
	}
	meta.Set(constants.ObjectTypeInventoryChangeLog, icl.Id, "item", item)

	qty := &apiresource.Quantity{
		ID:     icl.QuantityId,
		Object: constants.ObjectTypeQuantity,
		Value:  icl.QuantityValue,
		DisplayValue: apiresource.FormatDisplayValue(
			icl.QuantityValue,
			icl.QuantityUnitAbbreviation,
			icl.QuantityUnitType,
		),
		Unit: &apiresource.Unit{
			ID:                icl.QuantityUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              icl.QuantityUnitName,
			Abbreviation:      icl.QuantityUnitAbbreviation,
			Type:              constants.UnitType(icl.QuantityUnitType),
			RatioNumerator:    icl.QuantityUnitRatioNumerator,
			RatioDenominator:  icl.QuantityUnitRatioDenominator,
			OffsetNumerator:   icl.QuantityUnitOffsetNumerator,
			OffsetDenominator: icl.QuantityUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(icl.QuantityUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(icl.QuantityUnitUpdatedAt),
		},
	}
	meta.Set(constants.ObjectTypeInventoryChangeLog, icl.Id, "quantity", qty)

	if icl.ResponsibleUserId != nil {
		user := &apiresource.User{
			ID:     *icl.ResponsibleUserId,
			Object: constants.ObjectTypeUser,
			Name:   icl.ResponsibleUserName,
		}
		if icl.ResponsibleUserCreatedAt != nil {
			user.CreatedAt = icl.ResponsibleUserCreatedAt.AsTime()
		}
		if icl.ResponsibleUserUpdatedAt != nil {
			user.UpdatedAt = icl.ResponsibleUserUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeInventoryChangeLog, icl.Id, "responsible_user", user)
	}

	if icl.ScanningStationId != nil {
		meta.Set(constants.ObjectTypeInventoryChangeLog, icl.Id, "responsible_scanning_station_id", *icl.ScanningStationId)
	}
}
