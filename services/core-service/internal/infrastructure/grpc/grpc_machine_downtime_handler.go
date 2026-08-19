package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type machineDowntimeGRPCHandler struct {
	pb.UnimplementedCoreMachineDowntimeServiceServer

	machineDowntimeSvc domain.MachineDowntimeSvc
}

func machineDowntimeReasonToProto(r *domain.MachineDowntimeReason) *pb.MachineDowntimeReasonInfo {
	return &pb.MachineDowntimeReasonInfo{
		Id:        r.ID,
		Code:      r.Code,
		Name:      r.Name,
		OeeBucket: r.OeeBucket,
		IsPlanned: r.IsPlanned,
		SortOrder: r.SortOrder,
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

func machineDowntimeEventToProto(e *domain.MachineDowntimeEvent) *pb.MachineDowntimeEventInfo {
	info := &pb.MachineDowntimeEventInfo{
		Id:           e.ID,
		MachineId:    e.MachineID,
		ReasonCode:   e.ReasonCode,
		StartedAt:    timestamppb.New(e.StartedAt),
		ShiftDate:    timestamppb.New(e.ShiftDate),
		ReportedById: e.ReportedByID,
		SourceCode:   e.SourceCode,
		CreatedAt:    timestamppb.New(e.CreatedAt),
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
	}
	info.DepartmentId = e.DepartmentID
	info.ProductionStepId = e.ProductionStepID
	info.ReasonName = e.ReasonName
	info.ReasonOeeBucket = e.ReasonOeeBucket
	info.ReasonIsPlanned = e.ReasonIsPlanned
	info.ShiftCode = e.ShiftCode
	info.ItemId = e.ItemID
	info.ProductionRunId = e.ProductionRunID
	info.BatchId = e.BatchID
	info.ScheduleLineId = e.ScheduleLineID
	info.Note = e.Note
	if e.EndedAt != nil {
		info.EndedAt = timestamppb.New(*e.EndedAt)
	}
	if e.DurationSeconds != nil {
		info.DurationSeconds = e.DurationSeconds
	}
	return info
}

func downtimeTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func (h *machineDowntimeGRPCHandler) ListMachineDowntimeReasons(ctx context.Context, req *pb.ListMachineDowntimeReasonsRequest) (*pb.ListMachineDowntimeReasonsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	reasons, apiErr := h.machineDowntimeSvc.ListDowntimeReasons(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.MachineDowntimeReasonInfo, len(reasons))
	for i, r := range reasons {
		out[i] = machineDowntimeReasonToProto(r)
	}

	return &pb.ListMachineDowntimeReasonsResponse{Reasons: out}, nil
}

func (h *machineDowntimeGRPCHandler) ListMachineDowntimeEvents(ctx context.Context, req *pb.ListMachineDowntimeEventsRequest) (*pb.ListMachineDowntimeEventsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListMachineDowntimeEventsParams{
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		MachineIDs:    req.MachineIds,
		DepartmentIDs: req.DepartmentIds,
		ReasonCodes:   req.ReasonCodes,
		OpenOnly:      req.OpenOnly,
		Query:         req.Query,
	}

	if req.StartDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for starts_at.", "starts_at"))
		}
		params.StartDate = &parsed
	}
	if req.EndDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("Invalid date format for ends_at.", "ends_at"))
		}
		params.EndDate = &parsed
	}

	result, apiErr := h.machineDowntimeSvc.ListDowntimeEvents(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	events := make([]*pb.MachineDowntimeEventInfo, len(result.Events))
	for i, e := range result.Events {
		events[i] = machineDowntimeEventToProto(e)
	}

	return &pb.ListMachineDowntimeEventsResponse{
		Events: events,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *machineDowntimeGRPCHandler) GetMachineDowntimeEvent(ctx context.Context, req *pb.GetMachineDowntimeEventRequest) (*pb.GetMachineDowntimeEventResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	event, apiErr := h.machineDowntimeSvc.GetDowntimeEvent(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetMachineDowntimeEventResponse{Event: machineDowntimeEventToProto(event)}, nil
}

func (h *machineDowntimeGRPCHandler) CreateMachineDowntimeEvent(ctx context.Context, req *pb.CreateMachineDowntimeEventRequest) (*pb.CreateMachineDowntimeEventResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if req.StartedAt == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationErrorWithParam("started_at is required.", "started_at"))
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateMachineDowntimeEventParams{
		MachineID:       req.MachineId,
		ReasonCode:      req.ReasonCode,
		StartedAt:       req.StartedAt.AsTime(),
		EndedAt:         downtimeTimePtr(req.EndedAt),
		ItemID:          req.ItemId,
		ProductionRunID: req.ProductionRunId,
		BatchID:         req.BatchId,
		Note:            req.Note,
		SourceCode:      req.SourceCode,
		Duration:        downtimeDurationFromProto(req.DurationValue, req.DurationUnitId),
	}

	event, apiErr := h.machineDowntimeSvc.CreateDowntimeEvent(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateMachineDowntimeEventResponse{Event: machineDowntimeEventToProto(event)}, nil
}

func (h *machineDowntimeGRPCHandler) UpdateMachineDowntimeEvent(ctx context.Context, req *pb.UpdateMachineDowntimeEventRequest) (*pb.UpdateMachineDowntimeEventResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateMachineDowntimeEventParams{
		EventID:         req.Id,
		ReasonCode:      req.ReasonCode,
		StartedAt:       downtimeTimePtr(req.StartedAt),
		EndedAt:         field.TimestampClearableFromProto(req.EndedAt),
		ItemID:          field.StringClearableFromProto(req.ItemId),
		ProductionRunID: field.StringClearableFromProto(req.ProductionRunId),
		BatchID:         field.StringClearableFromProto(req.BatchId),
		Note:            field.StringClearableFromProto(req.Note),
		MachineID:       req.MachineId,
		Duration:        downtimeDurationClearableFromProto(req.Duration),
	}

	event, apiErr := h.machineDowntimeSvc.UpdateDowntimeEvent(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateMachineDowntimeEventResponse{Event: machineDowntimeEventToProto(event)}, nil
}

func (h *machineDowntimeGRPCHandler) DeleteMachineDowntimeEvent(ctx context.Context, req *pb.DeleteMachineDowntimeEventRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.machineDowntimeSvc.DeleteDowntimeEvent(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *machineDowntimeGRPCHandler) BatchGetMachineDowntimeEventsByIDs(ctx context.Context, req *pb.BatchGetMachineDowntimeEventsByIDsRequest) (*pb.BatchGetMachineDowntimeEventsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	events, apiErr := h.machineDowntimeSvc.BatchGetDowntimeEventsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.MachineDowntimeEventInfo, len(events))
	for i, e := range events {
		out[i] = machineDowntimeEventToProto(e)
	}

	return &pb.BatchGetMachineDowntimeEventsByIDsResponse{Events: out}, nil
}

// downtimeDurationFromProto reads the flat value/unit pair a create sends. Half a quantity is not a duration, so a value without its unit is dropped rather than defaulted to seconds.
func downtimeDurationFromProto(value, unitID *string) *domain.DowntimeDurationInput {
	if value == nil || unitID == nil || *value == "" || *unitID == "" {
		return nil
	}
	return &domain.DowntimeDurationInput{Value: *value, UnitID: *unitID}
}

// downtimeDurationClearableFromProto maps the update's QuantityPatch, where clear means "reopen this event" for the same reason a cleared ended_at does.
func downtimeDurationClearableFromProto(patch *pb.QuantityPatch) field.Clearable[domain.DowntimeDurationInput] {
	if patch == nil {
		return field.Unset[domain.DowntimeDurationInput]()
	}
	if patch.Clear {
		return field.Clear[domain.DowntimeDurationInput]()
	}
	if patch.Value == nil || patch.UnitId == nil {
		return field.Unset[domain.DowntimeDurationInput]()
	}
	return field.Set(domain.DowntimeDurationInput{Value: *patch.Value, UnitID: *patch.UnitId})
}
