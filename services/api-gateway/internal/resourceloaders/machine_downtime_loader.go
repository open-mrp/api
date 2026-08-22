package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var machineDowntimeLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.machine_downtime")

func LoadMachineDowntimeEvents(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, machineDowntimeLoaderTracer, "loader.machine_downtime_events.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMachineDowntimeEventsByIDsResponse, error) {
			return machineDowntimeClient.BatchGetMachineDowntimeEventsByIDs(ctx, &pb.BatchGetMachineDowntimeEventsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Events))
	for _, e := range resp.Events {
		out[e.Id] = MachineDowntimeEventFromProto(e)
		stashDowntimeRefs(ctx, meta, e)
	}
	return out, nil
}

func MachineDowntimeEventFromProto(e *pb.MachineDowntimeEventInfo) *apiresource.MachineDowntimeEvent {
	out := &apiresource.MachineDowntimeEvent{
		ID:     e.Id,
		Object: constants.ObjectTypeMachineDowntimeEvent,
		Reason: &apiresource.MachineDowntimeReasonSummary{
			Object:    constants.ObjectTypeMachineDowntimeReason,
			Code:      constants.MachineDowntimeReasonCode(e.ReasonCode),
			Name:      e.ReasonName,
			OeeBucket: (*constants.OeeBucket)(e.ReasonOeeBucket),
		},
		StartedAt: grpcutil.TimestampToTime(e.StartedAt),
		ShiftDate: grpcutil.TimestampToTime(e.ShiftDate),
		ShiftCode: e.ShiftCode,
		Note:      e.Note,
		Source:    constants.MachineDowntimeSource(e.SourceCode),
		CreatedAt: grpcutil.TimestampToTime(e.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(e.UpdatedAt),
	}
	if e.ProductionRunId != nil && *e.ProductionRunId != "" {
		out.ProductionRun = apiresource.NewEntity(*e.ProductionRunId, constants.ObjectTypeProductionRun, nil, nil)
	}
	if e.BatchId != nil && *e.BatchId != "" {
		out.Batch = apiresource.NewEntity(*e.BatchId, constants.ObjectTypeBatch, nil, nil)
	}
	if e.ScheduleLineId != nil && *e.ScheduleLineId != "" {
		out.ScheduleLine = apiresource.NewEntity(*e.ScheduleLineId, constants.ObjectTypeProductionScheduleLine, nil, nil)
	}
	out.EndedAt = grpcutil.TimestampToTimePtr(e.EndedAt)
	out.DurationSeconds = e.DurationSeconds
	return out
}

func stashDowntimeRefs(ctx context.Context, meta *resourcekit.LoadMeta, e *pb.MachineDowntimeEventInfo) {
	if e.MachineId != "" {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, e.Id, "machine_id", e.MachineId)
	}
	if e.DepartmentId != nil && *e.DepartmentId != "" {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, e.Id, "department_id", *e.DepartmentId)
	}
	if e.ItemId != nil && *e.ItemId != "" {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, e.Id, "item_id", *e.ItemId)
	}
	// The reporter is stored as a bare identity-actor id, so the Actor is built here from its prefix and preheated — LoadActors has no backing store.
	if actor := ActorRefFromID(e.ReportedById); actor != nil {
		meta.Set(constants.ObjectTypeMachineDowntimeEvent, e.Id, "reported_by_id", actor.ID)
		resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
	}
}
