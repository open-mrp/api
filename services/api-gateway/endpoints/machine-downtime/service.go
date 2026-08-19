package machinedowntimeep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MachineDowntimeSvc interface {
	ListDowntimeReasons(ctx context.Context, req *ListMachineDowntimeReasonsRequest) (*apiresource.List[apiresource.MachineDowntimeReason], *apierror.APIError)
	ListDowntimeEvents(ctx context.Context, req *ListMachineDowntimeEventsRequest) (*apiresource.List[apiresource.MachineDowntimeEvent], *apierror.APIError)
	GetDowntimeEvent(ctx context.Context, req *RetrieveMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError)
	CreateDowntimeEvent(ctx context.Context, req *CreateMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError)
	UpdateDowntimeEvent(ctx context.Context, req *UpdateMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError)
	DeleteDowntimeEvent(ctx context.Context, req *DeleteMachineDowntimeEventRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type MachineDowntimeSvcConfig struct {
	// CoreClient (required) is the core-service machine-downtime gRPC client.
	CoreClient pb.CoreMachineDowntimeServiceClient
}

type machineDowntimeSvcImpl struct {
	coreClient pb.CoreMachineDowntimeServiceClient
}

var machineDowntimeEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.machine-downtime.service")

func (c *MachineDowntimeSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("machine downtime endpoint service: core client is required")
	}
	return nil
}

func NewMachineDowntimeSvc(config *MachineDowntimeSvcConfig) MachineDowntimeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &machineDowntimeSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *machineDowntimeSvcImpl) ListDowntimeReasons(ctx context.Context, req *ListMachineDowntimeReasonsRequest) (*apiresource.List[apiresource.MachineDowntimeReason], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, machineDowntimeEpSvcTracer, "service.machine_downtime.list_reasons", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMachineDowntimeReasonsResponse, error) {
			return m.coreClient.ListMachineDowntimeReasons(ctx, &pb.ListMachineDowntimeReasonsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	reasons := make([]apiresource.MachineDowntimeReason, len(resp.Reasons))
	for i, r := range resp.Reasons {
		reasons[i] = MachineDowntimeReasonFromProto(r)
	}

	return apiresource.NewList(reasons, apiresource.PageInfo{}), nil
}

func (m *machineDowntimeSvcImpl) ListDowntimeEvents(ctx context.Context, req *ListMachineDowntimeEventsRequest) (*apiresource.List[apiresource.MachineDowntimeEvent], *apierror.APIError) {
	pbReq := &pb.ListMachineDowntimeEventsRequest{
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		MachineIds:    req.MachineIDs,
		DepartmentIds: req.DepartmentIDs,
		ReasonCodes:   downtimeEnumStrings(req.Reasons),
		OpenOnly:      req.Open,
		Query:         req.Query,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineDowntimeEpSvcTracer, "service.machine_downtime.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListMachineDowntimeEventsResponse, error) {
			return m.coreClient.ListMachineDowntimeEvents(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	events := make([]apiresource.MachineDowntimeEvent, len(resp.Events))
	reporters := make([]*apiresource.Actor, 0, len(resp.Events))
	for i, e := range resp.Events {
		events[i] = MachineDowntimeEventFromProto(e)
		reporters = append(reporters, StashMachineDowntimeEventMeta(ctx, meta, e))
	}
	hydrateReporters(ctx, reporters)

	return apiresource.NewList(events, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *machineDowntimeSvcImpl) GetDowntimeEvent(ctx context.Context, req *RetrieveMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
	pbReq := &pb.GetMachineDowntimeEventRequest{Id: req.MachineDowntimeEventID}

	resp, apiErr := grpcutil.CallRPC(ctx, machineDowntimeEpSvcTracer, "service.machine_downtime.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetMachineDowntimeEventResponse, error) {
			return m.coreClient.GetMachineDowntimeEvent(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := MachineDowntimeEventFromProto(resp.Event)
	hydrateReporters(ctx, []*apiresource.Actor{StashMachineDowntimeEventMeta(ctx, meta, resp.Event)})
	return &result, nil
}

func (m *machineDowntimeSvcImpl) CreateDowntimeEvent(ctx context.Context, req *CreateMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
	pbReq := &pb.CreateMachineDowntimeEventRequest{
		MachineId:       req.MachineID,
		ReasonCode:      string(req.Reason),
		StartedAt:       timestamppb.New(req.StartedAt),
		ItemId:          req.ItemID.Ptr(),
		ProductionRunId: req.ProductionRunID.Ptr(),
		BatchId:         req.BatchID.Ptr(),
		Note:            req.Note.Ptr(),
		SourceCode:      downtimeEnumPtr(req.Source.Ptr()),
	}
	if endedAt := req.EndedAt.Ptr(); endedAt != nil {
		pbReq.EndedAt = timestamppb.New(*endedAt)
	}
	if duration := req.Duration.Ptr(); duration != nil {
		pbReq.DurationValue = &duration.Value
		pbReq.DurationUnitId = &duration.UnitID
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineDowntimeEpSvcTracer, "service.machine_downtime.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateMachineDowntimeEventResponse, error) {
			return m.coreClient.CreateMachineDowntimeEvent(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := MachineDowntimeEventFromProto(resp.Event)
	hydrateReporters(ctx, []*apiresource.Actor{StashMachineDowntimeEventMeta(ctx, meta, resp.Event)})
	return &result, nil
}

func (m *machineDowntimeSvcImpl) UpdateDowntimeEvent(ctx context.Context, req *UpdateMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
	pbReq := &pb.UpdateMachineDowntimeEventRequest{
		Id:         req.MachineDowntimeEventID,
		ReasonCode: downtimeEnumPtr(req.Reason.Ptr()),
		// Clearable nullable fields → *Patch (clear / set / leave). A cleared ended_at reopens the event.
		EndedAt:         field.TimestampClearableToProto(req.EndedAt),
		ItemId:          field.StringClearableToProto(req.ItemID),
		ProductionRunId: field.StringClearableToProto(req.ProductionRunID),
		BatchId:         field.StringClearableToProto(req.BatchID),
		Note:            field.StringClearableToProto(req.Note),
		MachineId:       req.MachineID.Ptr(),
		Duration:        apirequest.QuantityFieldToProto(req.Duration),
	}
	if startedAt := req.StartedAt.Ptr(); startedAt != nil {
		pbReq.StartedAt = timestamppb.New(*startedAt)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, machineDowntimeEpSvcTracer, "service.machine_downtime.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateMachineDowntimeEventResponse, error) {
			return m.coreClient.UpdateMachineDowntimeEvent(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := MachineDowntimeEventFromProto(resp.Event)
	hydrateReporters(ctx, []*apiresource.Actor{StashMachineDowntimeEventMeta(ctx, meta, resp.Event)})
	return &result, nil
}

func (m *machineDowntimeSvcImpl) DeleteDowntimeEvent(ctx context.Context, req *DeleteMachineDowntimeEventRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteMachineDowntimeEventRequest{Id: req.MachineDowntimeEventID}

	_, apiErr := grpcutil.CallRPC(ctx, machineDowntimeEpSvcTracer, "service.machine_downtime.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteMachineDowntimeEvent(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

// MachineDowntimeReasonFromProto maps a core MachineDowntimeReasonInfo to the API resource.
func MachineDowntimeReasonFromProto(info *pb.MachineDowntimeReasonInfo) apiresource.MachineDowntimeReason {
	return apiresource.MachineDowntimeReason{
		ID:             info.Id,
		Object:         constants.ObjectTypeMachineDowntimeReason,
		Code:           constants.MachineDowntimeReasonCode(info.Code),
		Name:           info.Name,
		OeeBucket:      constants.OeeBucket(info.OeeBucket),
		PlanningStatus: constants.DowntimePlanningStatusOf(info.IsPlanned),
		SortOrder:      info.SortOrder,
		CreatedAt:      grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(info.UpdatedAt),
	}
}

// MachineDowntimeEventFromProto maps a core MachineDowntimeEventInfo to the API resource. The machine, department, item and reported_by expandables are left nil; pair with StashMachineDowntimeEventMeta so they resolve on ?include=.
func MachineDowntimeEventFromProto(info *pb.MachineDowntimeEventInfo) apiresource.MachineDowntimeEvent {
	e := apiresource.MachineDowntimeEvent{
		ID:     info.Id,
		Object: constants.ObjectTypeMachineDowntimeEvent,
		Reason: &apiresource.MachineDowntimeReasonSummary{
			Object:    constants.ObjectTypeMachineDowntimeReason,
			Code:      constants.MachineDowntimeReasonCode(info.ReasonCode),
			Name:      info.ReasonName,
			OeeBucket: (*constants.OeeBucket)(info.ReasonOeeBucket),
		},
		StartedAt: grpcutil.TimestampToTime(info.StartedAt),
		ShiftDate: grpcutil.TimestampToTime(info.ShiftDate),
		ShiftCode: info.ShiftCode,
		Note:      info.Note,
		Source:    constants.MachineDowntimeSource(info.SourceCode),
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
	if info.ProductionRunId != nil && *info.ProductionRunId != "" {
		e.ProductionRun = apiresource.NewEntity(*info.ProductionRunId, constants.ObjectTypeProductionRun, nil, nil)
	}
	if info.BatchId != nil && *info.BatchId != "" {
		e.Batch = apiresource.NewEntity(*info.BatchId, constants.ObjectTypeBatch, nil, nil)
	}
	if info.ScheduleLineId != nil && *info.ScheduleLineId != "" {
		e.ScheduleLine = apiresource.NewEntity(*info.ScheduleLineId, constants.ObjectTypeProductionScheduleLine, nil, nil)
	}
	e.EndedAt = grpcutil.TimestampToTimePtr(info.EndedAt)
	e.DurationSeconds = info.DurationSeconds
	return e
}

// StashMachineDowntimeEventMeta stashes the FK ids in LoadMeta so the loaders can resolve the expandable machine, department, item and reported_by on ?include=. The reporter is stored as a bare identity-actor id, so its Actor is built from the id's prefix and preheated — LoadActors has no backing store. The built actor is returned (nil when absent) so the caller can batch-hydrate display names.
func StashMachineDowntimeEventMeta(ctx context.Context, meta *resourcekit.LoadMeta, info *pb.MachineDowntimeEventInfo) *apiresource.Actor {
	if info == nil {
		return nil
	}
	if info.MachineId != "" {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, info.Id, "machine_id", info.MachineId)
	}
	if info.DepartmentId != nil && *info.DepartmentId != "" {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, info.Id, "department_id", *info.DepartmentId)
	}
	if info.ItemId != nil && *info.ItemId != "" {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, info.Id, "item_id", *info.ItemId)
	}
	actor := resourceloaders.ActorRefFromID(info.ReportedById)
	if actor != nil {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, info.Id, "reported_by_id", actor.ID)
		resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
	}
	return actor
}

// hydrateReporters fills the reporters' display names + handles. A no-op unless the caller expanded reported_by — the only case where the names are rendered — avoiding needless loader round-trips otherwise.
func hydrateReporters(ctx context.Context, reporters []*apiresource.Actor) {
	if !resourcekit.RequestedIncludeSet(ctx)["reported_by"] {
		return
	}
	resourceloaders.HydrateIdentityActorNames(ctx, reporters)
}

// downtimeEnumPtr narrows a typed-enum pointer to the plain string pointer the proto layer uses. The storage vocabulary stays untyped; the typing lives at the API boundary.
func downtimeEnumPtr[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

// downtimeEnumStrings narrows a typed-enum slice to the plain strings the proto layer uses.
func downtimeEnumStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
