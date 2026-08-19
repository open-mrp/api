package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func operatingCalendarToProto(c *domain.OperatingCalendar) *pb.OperatingCalendarInfo {
	if c == nil {
		return nil
	}
	return &pb.OperatingCalendarInfo{
		Id:         c.ID,
		AccountId:  c.AccountID,
		Code:       c.Code,
		Name:       c.Name,
		KindCode:   c.KindCode,
		DaysOfWeek: c.DaysOfWeek,
		CutoffAt:   c.CutoffAt,
		Timezone:   c.Timezone,
		IsDefault:  c.IsDefault,
		CreatedAt:  timestamppb.New(c.CreatedAt),
		UpdatedAt:  timestamppb.New(c.UpdatedAt),
	}
}

func operatingCalendarClosureToProto(c domain.OperatingCalendarClosure) *pb.OperatingCalendarClosureInfo {
	return &pb.OperatingCalendarClosureInfo{
		Id:                  c.ID,
		OperatingCalendarId: c.CalendarID,
		ClosedOn:            timestamppb.New(c.ClosedOn),
		Name:                c.Name,
		CreatedAt:           timestamppb.New(c.CreatedAt),
		UpdatedAt:           timestamppb.New(c.UpdatedAt),
	}
}

func (h *productionScheduleGRPCHandler) ListOperatingCalendars(ctx context.Context, req *pb.ListOperatingCalendarsRequest) (*pb.ListOperatingCalendarsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	calendars, apiErr := h.operatingCalendarSvc.ListOperatingCalendars(ctx, req.KindCode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := &pb.ListOperatingCalendarsResponse{Calendars: make([]*pb.OperatingCalendarInfo, 0, len(calendars))}
	for _, c := range calendars {
		out.Calendars = append(out.Calendars, operatingCalendarToProto(&c))
	}
	return out, nil
}

func (h *productionScheduleGRPCHandler) GetOperatingCalendar(ctx context.Context, req *pb.GetOperatingCalendarRequest) (*pb.OperatingCalendarInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	calendar, apiErr := h.operatingCalendarSvc.GetOperatingCalendar(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatingCalendarToProto(calendar), nil
}

func (h *productionScheduleGRPCHandler) CreateOperatingCalendar(ctx context.Context, req *pb.CreateOperatingCalendarRequest) (*pb.OperatingCalendarInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	calendar, apiErr := h.operatingCalendarSvc.CreateOperatingCalendar(ctx, domain.CreateOperatingCalendarParams{
		Code:       req.Code,
		Name:       req.Name,
		KindCode:   req.KindCode,
		DaysOfWeek: req.DaysOfWeek,
		CutoffAt:   req.CutoffAt,
		Timezone:   req.Timezone,
		IsDefault:  req.IsDefault,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatingCalendarToProto(calendar), nil
}

func (h *productionScheduleGRPCHandler) UpdateOperatingCalendar(ctx context.Context, req *pb.UpdateOperatingCalendarRequest) (*pb.OperatingCalendarInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.UpdateOperatingCalendarParams{
		ID:         req.Id,
		Name:       req.Name,
		DaysOfWeek: req.DaysOfWeek,
		IsDefault:  req.IsDefault,
	}

	// A clearable arrives as three states, and the repo needs the clear flag separately because a nil value already means "leave alone".
	cutoff := field.StringClearableFromProto(req.CutoffAt)
	if v, ok := cutoff.Value(); ok {
		params.CutoffAt = &v
	}
	params.ClearCutoffAt = cutoff.IsClear()

	timezone := field.StringClearableFromProto(req.Timezone)
	if v, ok := timezone.Value(); ok {
		params.Timezone = &v
	}
	params.ClearTimezone = timezone.IsClear()

	calendar, apiErr := h.operatingCalendarSvc.UpdateOperatingCalendar(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatingCalendarToProto(calendar), nil
}

func (h *productionScheduleGRPCHandler) DeleteOperatingCalendar(ctx context.Context, req *pb.DeleteOperatingCalendarRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.operatingCalendarSvc.DeleteOperatingCalendar(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func (h *productionScheduleGRPCHandler) ListOperatingCalendarClosures(ctx context.Context, req *pb.ListOperatingCalendarClosuresRequest) (*pb.ListOperatingCalendarClosuresResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var from, to *time.Time
	if req.FromDate != nil {
		t := req.FromDate.AsTime()
		from = &t
	}
	if req.ToDate != nil {
		t := req.ToDate.AsTime()
		to = &t
	}

	closures, apiErr := h.operatingCalendarSvc.ListOperatingCalendarClosures(ctx, req.OperatingCalendarId, from, to)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := &pb.ListOperatingCalendarClosuresResponse{Closures: make([]*pb.OperatingCalendarClosureInfo, 0, len(closures))}
	for _, c := range closures {
		out.Closures = append(out.Closures, operatingCalendarClosureToProto(c))
	}
	return out, nil
}

func (h *productionScheduleGRPCHandler) CreateOperatingCalendarClosure(ctx context.Context, req *pb.CreateOperatingCalendarClosureRequest) (*pb.OperatingCalendarClosureInfo, error) {
	if req == nil || req.ClosedOn == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	closure, apiErr := h.operatingCalendarSvc.CreateOperatingCalendarClosure(ctx, req.OperatingCalendarId, req.ClosedOn.AsTime(), req.Name)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return operatingCalendarClosureToProto(*closure), nil
}

func (h *productionScheduleGRPCHandler) DeleteOperatingCalendarClosure(ctx context.Context, req *pb.DeleteOperatingCalendarClosureRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if apiErr := h.operatingCalendarSvc.DeleteOperatingCalendarClosure(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}
