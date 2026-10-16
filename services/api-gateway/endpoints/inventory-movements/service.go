package inventorymovementep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/open-mrp/apikit/include"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

// InventoryMovementSvc serves inventory movements, inventory levels and lot-size defaults.
type InventoryMovementSvc interface {
	ListInventoryMovements(ctx context.Context, req *ListInventoryMovementsRequest) (*apiresource.List[apiresource.InventoryMovement], *apierror.APIError)
	GetInventoryMovement(ctx context.Context, req *RetrieveInventoryMovementRequest) (*apiresource.InventoryMovement, *apierror.APIError)
	CreateInventoryMovement(ctx context.Context, req *CreateInventoryMovementRequest) (*apiresource.InventoryMovement, *apierror.APIError)
	BulkCreateInventoryMovements(ctx context.Context, req *BulkCreateInventoryMovementsRequest) (*apiresource.AsyncJob, *apierror.APIError)
	ExportInventoryMovements(ctx context.Context, req *ExportInventoryMovementsRequest) (*apiresource.AsyncJob, *apierror.APIError)
	ListInventoryLevels(ctx context.Context, req *ListInventoryLevelsRequest) (*apiresource.List[apiresource.InventoryLevel], *apierror.APIError)
	GetItemInventoryLevel(ctx context.Context, req *RetrieveItemInventoryLevelRequest) (*apiresource.InventoryLevel, *apierror.APIError)
	GetLotSizeDefault(ctx context.Context, req *RetrieveLotSizeDefaultRequest) (*apiresource.LotSizeDefault, *apierror.APIError)
}

type InventoryMovementSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client for movement reads, exports and lot sizes.
	CoreClient pb.CoreServiceClient
	// InventoryClient (required) is the core-service inventory gRPC client for movement writes and levels.
	InventoryClient pb.CoreInventoryServiceClient
}

func (c *InventoryMovementSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("inventory movement endpoint service: core client is required")
	}
	if c.InventoryClient == nil {
		return fmt.Errorf("inventory movement endpoint service: inventory client is required")
	}
	return nil
}

type inventoryMovementSvcImpl struct {
	coreClient      pb.CoreServiceClient
	inventoryClient pb.CoreInventoryServiceClient
}

var tracer = tracing.GetTracer("api-gateway.endpoints.inventory_movements.service")

func NewInventoryMovementSvc(config *InventoryMovementSvcConfig) InventoryMovementSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &inventoryMovementSvcImpl{coreClient: config.CoreClient, inventoryClient: config.InventoryClient}
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func (m *inventoryMovementSvcImpl) ListInventoryMovements(ctx context.Context, req *ListInventoryMovementsRequest) (*apiresource.List[apiresource.InventoryMovement], *apierror.APIError) {
	pbReq := &pb.ListInventoryChangeLogsRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		ItemIds:     req.ItemIDs,
		Types:       stringsOf(req.Types),
		ActorIds:    req.ActorIDs,
		ActorTypes:  stringsOf(req.ActorTypes),
		LocationIds: req.LocationIDs,
	}
	if req.OccurredAfter != nil {
		pbReq.StartDate = timestamppb.New(*req.OccurredAfter)
	}
	if req.OccurredBefore != nil {
		pbReq.EndDate = timestamppb.New(*req.OccurredBefore)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_movements.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInventoryChangeLogsResponse, error) {
			return m.coreClient.ListInventoryChangeLogs(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.InventoryMovement, len(resp.InventoryChangeLogs))
	actors := make([]*apiresource.ActorRef, len(items))
	for i, info := range resp.InventoryChangeLogs {
		items[i] = MovementFromProto(ctx, info)
		actors[i] = items[i].Actor
	}
	resourceloaders.ResolveActorRefs(ctx, actors)
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *inventoryMovementSvcImpl) GetInventoryMovement(ctx context.Context, req *RetrieveInventoryMovementRequest) (*apiresource.InventoryMovement, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_movements.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetInventoryChangeLogResponse, error) {
			return m.coreClient.GetInventoryChangeLog(ctx, &pb.GetInventoryChangeLogRequest{Id: req.InventoryMovementID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	movement := MovementFromProto(ctx, resp.InventoryChangeLog)
	resourceloaders.ResolveActorRefs(ctx, []*apiresource.ActorRef{movement.Actor})
	return &movement, nil
}

func (m *inventoryMovementSvcImpl) CreateInventoryMovement(ctx context.Context, req *CreateInventoryMovementRequest) (*apiresource.InventoryMovement, *apierror.APIError) {
	pbReq := &pb.CreateInventoryMovementRequest{
		ItemId:     req.ItemID,
		Reconcile:  req.Operation == constants.InventoryUpdateOperationReconcile,
		Measure:    req.Quantity.Value,
		UnitId:     req.Quantity.UnitID,
		LocationId: req.LocationID.Ptr(),
		LotId:      req.LotID.Ptr(),
		LotNumber:  req.LotNumber.Ptr(),
		CustomerId: req.CustomerID.Ptr(),
		Reason:     req.Reason.Ptr().StringPtr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_movements.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateInventoryMovementResponse, error) {
			return m.inventoryClient.CreateInventoryMovement(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	movement := MovementFromProto(ctx, resp.Movement)
	resourceloaders.ResolveActorRefs(ctx, []*apiresource.ActorRef{movement.Actor})
	return &movement, nil
}

func (m *inventoryMovementSvcImpl) BulkCreateInventoryMovements(ctx context.Context, req *BulkCreateInventoryMovementsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	rows := make([]*pb.BulkCreateInventoryMovementRow, len(req.Movements))
	for i, r := range req.Movements {
		rows[i] = &pb.BulkCreateInventoryMovementRow{
			ItemId:     r.ItemID.Ptr(),
			Sku:        r.SKU.Ptr(),
			Measure:    r.Quantity.Value,
			UnitId:     r.Quantity.UnitID,
			LocationId: r.LocationID.Ptr(),
			LotNumber:  r.LotNumber.Ptr(),
			Reason:     r.Reason.Ptr().StringPtr(),
		}
	}
	pbReq := &pb.BulkCreateInventoryMovementsRequest{Reconcile: req.Operation == constants.InventoryUpdateOperationReconcile, Rows: rows}
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_movements.bulk_create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkCreateInventoryMovementsResponse, error) {
			return m.inventoryClient.BulkCreateInventoryMovements(ctx, pbReq, opts...)
		}, grpcutil.WithTimeout(grpcutil.BulkWriteTimeout))
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

func (m *inventoryMovementSvcImpl) ExportInventoryMovements(ctx context.Context, req *ExportInventoryMovementsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	pbReq := &pb.StartInventoryChangeLogsExportRequest{
		ItemIds:     req.ItemIDs,
		Types:       stringsOf(req.Types),
		ActorIds:    req.ActorIDs,
		ActorTypes:  stringsOf(req.ActorTypes),
		LocationIds: req.LocationIDs,
	}
	if v, ok := req.OccurredAfter.Value(); ok {
		pbReq.StartDate = timestamppb.New(v)
	}
	if v, ok := req.OccurredBefore.Value(); ok {
		pbReq.EndDate = timestamppb.New(v)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_movements.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.StartInventoryChangeLogsExportResponse, error) {
			return m.coreClient.StartInventoryChangeLogsExport(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

func (m *inventoryMovementSvcImpl) ListInventoryLevels(ctx context.Context, req *ListInventoryLevelsRequest) (*apiresource.List[apiresource.InventoryLevel], *apierror.APIError) {
	pbReq := &pb.ListInventoryLevelsRequest{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		ItemIds:     req.ItemIDs,
		LocationIds: req.LocationIDs,
		SiteIds:     req.SiteIDs,
	}
	if req.AsOf != nil {
		pbReq.AsOf = timestamppb.New(*req.AsOf)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_levels.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInventoryLevelsResponse, error) {
			return m.inventoryClient.ListInventoryLevels(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	levels := make([]apiresource.InventoryLevel, len(resp.Levels))
	for i, l := range resp.Levels {
		levels[i] = LevelFromProto(l)
	}
	return apiresource.NewList(levels, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *inventoryMovementSvcImpl) GetItemInventoryLevel(ctx context.Context, req *RetrieveItemInventoryLevelRequest) (*apiresource.InventoryLevel, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.inventory_levels.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInventoryLevelsResponse, error) {
			return m.inventoryClient.ListInventoryLevels(ctx, &pb.ListInventoryLevelsRequest{Limit: 1, ItemIds: []string{req.ItemID}}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(resp.Levels) == 0 {
		return nil, apierror.NewNotFoundError("No stocked item has that ID.")
	}
	level := LevelFromProto(resp.Levels[0])
	return &level, nil
}

func (m *inventoryMovementSvcImpl) GetLotSizeDefault(ctx context.Context, req *RetrieveLotSizeDefaultRequest) (*apiresource.LotSizeDefault, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, tracer, "service.lot_size_defaults.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetItemLotDefaultResponse, error) {
			return m.coreClient.GetItemLotDefault(ctx, &pb.GetItemLotDefaultRequest{ItemId: req.ItemID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return LotSizeDefaultFromProto(resp), nil
}

// MovementFromProto presents a movement and stashes its item, location, lot and source document for their includes.
func MovementFromProto(ctx context.Context, info *pb.InventoryChangeLogInfo) apiresource.InventoryMovement {
	if info == nil {
		return apiresource.InventoryMovement{}
	}
	created := grpcutil.TimestampToTime(info.CreatedAt)
	mv := apiresource.InventoryMovement{
		ID:         info.Id,
		Object:     constants.ObjectTypeInventoryMovement,
		Type:       constants.InventoryMovementType(info.Type),
		Quantity:   apiresource.NewQuantity(apiresource.NormalizeQuantityValue(info.QuantityValue, info.QuantityUnitType), info.QuantityUnitId),
		Actor:      resourceloaders.ActorRefFromProto(info.Actor),
		UnitCost:   resourceloaders.UnitPriceFromProto(info.UnitCost),
		OccurredAt: created,
		CreatedAt:  created,
	}
	if info.Reason != nil {
		reason := constants.InventoryMovementReason(*info.Reason)
		mv.Reason = &reason
	}
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeInventoryMovement, info.Id, "item_id", info.ItemId)
	if info.LocationId != nil {
		meta.Set(constants.ObjectTypeInventoryMovement, info.Id, "location_id", *info.LocationId)
	}
	if info.LotId != nil {
		meta.Set(constants.ObjectTypeInventoryMovement, info.Id, "lot", &apiresource.Lot{
			ID: *info.LotId, Object: constants.ObjectTypeLot, LotNumber: info.GetLotNumber(),
		})
	}
	if info.RelatedType != nil && info.RelatedId != nil {
		meta.Set(constants.ObjectTypeInventoryMovement, info.Id, "related",
			recordstub.Ref{Type: constants.RecordType(*info.RelatedType), ID: *info.RelatedId})
	}
	return mv
}

func optionalQuantity(value *string, unitID string) *apiresource.Quantity {
	if value == nil {
		return nil
	}
	return apiresource.NewQuantity(apiresource.NormalizeQuantityValue(*value, ""), unitID)
}

// LevelFromProto presents an inventory level, its item and location kept for their includes.
func LevelFromProto(l *pb.InventoryLevelInfo) apiresource.InventoryLevel {
	level := apiresource.InventoryLevel{
		Object:             constants.ObjectTypeInventoryLevel,
		OnHand:             optionalQuantity(&l.OnHand, l.UnitId),
		Reserved:           optionalQuantity(l.Reserved, l.UnitId),
		AvailableToPromise: optionalQuantity(l.AvailableToPromise, l.UnitId),
		Short:              optionalQuantity(l.Short, l.UnitId),
	}
	level.SetRefs(l.ItemId, l.LocationId)
	return level
}

// LotSizeDefaultFromProto presents an item's resolved lot size; a lot of zero is no lot convention.
func LotSizeDefaultFromProto(resp *pb.GetItemLotDefaultResponse) *apiresource.LotSizeDefault {
	d := &apiresource.LotSizeDefault{Object: constants.ObjectTypeLotSizeDefault}
	if resp.Quantity != 0 && resp.Source != "" {
		value := strconv.FormatFloat(resp.Quantity, 'f', -1, 64)
		d.LotSize = apiresource.NewQuantity(apiresource.NormalizeQuantityValue(value, ""), resp.GetUnitId())
		source := constants.ItemLotSource(resp.Source)
		d.Source = &source
	}
	d.SetRefs(resp.ItemId, resp.GetItemCategoryId())
	return d
}
