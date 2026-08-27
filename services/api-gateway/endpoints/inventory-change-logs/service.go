package inventorychangelogep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	"github.com/open-mrp/api/services/api-gateway/internal/export"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InventoryChangeLogSvc interface {
	ListInventoryChangeLogs(ctx context.Context, req *ListInventoryChangeLogsRequest) (*apiresource.List[apiresource.InventoryChangeLog], *apierror.APIError)
	GetInventoryChangeLog(ctx context.Context, req *RetrieveInventoryChangeLogRequest) (*apiresource.InventoryChangeLog, *apierror.APIError)
	ExportInventoryChangeLogs(ctx context.Context, req *ExportInventoryChangeLogsRequest) (*httptransport.FileDownload, *apierror.APIError)
}

type InventoryChangeLogSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		ActionTypeCodes:  actionTypeStrings(req.ActionTypes),
		ChangedByUserIds: req.ChangedByUserIDs,
	}

	if req.StartsAt != nil {
		pbReq.StartDate = timestamppb.New(*req.StartsAt)
	}
	if req.EndsAt != nil {
		pbReq.EndDate = timestamppb.New(*req.EndsAt)
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
		ActionTypeCodes:  actionTypeStrings(req.ActionTypes),
		ChangedByUserIds: req.ChangedByUserIDs,
	}

	if req.StartsAt != nil {
		pbReq.StartDate = timestamppb.New(*req.StartsAt)
	}
	if req.EndsAt != nil {
		pbReq.EndDate = timestamppb.New(*req.EndsAt)
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
	if req.StartsAt != nil {
		startDateStr = req.StartsAt.Format("2006-01-02")
	}
	endDateStr := "all"
	if req.EndsAt != nil {
		endDateStr = req.EndsAt.Format("2006-01-02")
	}
	filename := fmt.Sprintf("inventory-change-logs-%s-%s.xlsx", startDateStr, endDateStr)

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    filename,
		Body:        body,
	}, nil
}

// actionTypeStrings unwraps the typed filter for the proto request, which carries codes as plain strings.
func actionTypeStrings(types []constants.InventoryActionType) []string {
	out := make([]string, len(types))
	for i, t := range types {
		out[i] = string(t)
	}
	return out
}

func inventoryChangeLogFromProto(icl *pb.InventoryChangeLogInfo) apiresource.InventoryChangeLog {
	if icl == nil {
		return apiresource.InventoryChangeLog{}
	}

	return apiresource.InventoryChangeLog{
		ID:         icl.Id,
		Object:     constants.ObjectTypeInventoryChangeLog,
		ActionType: constants.InventoryActionType(icl.ActionTypeCode),
		Quantity:   inventoryChangeLogQuantity(icl),
		CreatedAt:  grpcutil.TimestampToTime(icl.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(icl.UpdatedAt),
	}
}

// inventoryChangeLogQuantity builds the signed amount the entry recorded. Carried inline rather than stashed for an include: the amount is what the entry is, and an entry reporting null for it says nothing.
func inventoryChangeLogQuantity(icl *pb.InventoryChangeLogInfo) *apiresource.Quantity {
	return &apiresource.Quantity{
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
