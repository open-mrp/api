package operatingcalendarep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OperatingCalendarSvc interface {
	ListOperatingCalendars(ctx context.Context, req *ListOperatingCalendarsRequest) (*apiresource.List[apiresource.OperatingCalendar], *apierror.APIError)
	RetrieveOperatingCalendar(ctx context.Context, req *RetrieveOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError)
	CreateOperatingCalendar(ctx context.Context, req *CreateOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError)
	UpdateOperatingCalendar(ctx context.Context, req *UpdateOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError)
	DeleteOperatingCalendar(ctx context.Context, req *DeleteOperatingCalendarRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListOperatingCalendarClosures(ctx context.Context, req *ListOperatingCalendarClosuresRequest) (*apiresource.List[apiresource.OperatingCalendarClosure], *apierror.APIError)
	CreateOperatingCalendarClosure(ctx context.Context, req *CreateOperatingCalendarClosureRequest) (*apiresource.OperatingCalendarClosure, *apierror.APIError)
	DeleteOperatingCalendarClosure(ctx context.Context, req *DeleteOperatingCalendarClosureRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type OperatingCalendarSvcConfig struct {
	// CoreClient (required) is the core-service production-schedule gRPC client, which carries the calendar RPCs.
	CoreClient pb.CoreProductionScheduleServiceClient
}

type operatingCalendarSvcImpl struct {
	coreClient pb.CoreProductionScheduleServiceClient
}

var operatingCalendarEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.operating-calendars.service")

func (c *OperatingCalendarSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("operating calendars endpoint service: core client is required")
	}
	return nil
}

func NewOperatingCalendarSvc(config *OperatingCalendarSvcConfig) OperatingCalendarSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &operatingCalendarSvcImpl{coreClient: config.CoreClient}
}

func calendarFromProto(info *pb.OperatingCalendarInfo) apiresource.OperatingCalendar {
	return apiresource.OperatingCalendar{
		Object:     constants.ObjectTypeOperatingCalendar,
		ID:         info.Id,
		Code:       info.Code,
		Name:       info.Name,
		Kind:       constants.OperatingCalendarKind(info.KindCode),
		DaysOfWeek: info.DaysOfWeek,
		CutoffAt:   info.CutoffAt,
		Timezone:   info.Timezone,
		IsDefault:  info.IsDefault,
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}
}

func closureFromProto(info *pb.OperatingCalendarClosureInfo) apiresource.OperatingCalendarClosure {
	return apiresource.OperatingCalendarClosure{
		Object:              constants.ObjectTypeOperatingCalendarClosure,
		ID:                  info.Id,
		OperatingCalendarID: info.OperatingCalendarId,
		ClosedOn:            grpcutil.TimestampToTime(info.ClosedOn),
		Name:                info.Name,
		CreatedAt:           grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(info.UpdatedAt),
	}
}

func (m *operatingCalendarSvcImpl) ListOperatingCalendars(ctx context.Context, req *ListOperatingCalendarsRequest) (*apiresource.List[apiresource.OperatingCalendar], *apierror.APIError) {
	pbReq := &pb.ListOperatingCalendarsRequest{}
	if req.Kind != nil {
		kind := string(*req.Kind)
		pbReq.KindCode = &kind
	}

	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOperatingCalendarsResponse, error) {
			return m.coreClient.ListOperatingCalendars(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	calendars := make([]apiresource.OperatingCalendar, len(resp.Calendars))
	for i, c := range resp.Calendars {
		calendars[i] = calendarFromProto(c)
	}
	return apiresource.NewList(calendars, apiresource.PageInfo{}), nil
}

func (m *operatingCalendarSvcImpl) RetrieveOperatingCalendar(ctx context.Context, req *RetrieveOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.retrieve", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatingCalendarInfo, error) {
			return m.coreClient.GetOperatingCalendar(ctx, &pb.GetOperatingCalendarRequest{Id: req.OperatingCalendarID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := calendarFromProto(resp)
	return &out, nil
}

func (m *operatingCalendarSvcImpl) CreateOperatingCalendar(ctx context.Context, req *CreateOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
	pbReq := &pb.CreateOperatingCalendarRequest{
		Code:       req.Code,
		Name:       req.Name,
		KindCode:   string(req.Kind),
		DaysOfWeek: req.DaysOfWeek,
		CutoffAt:   req.CutoffAt.Ptr(),
		Timezone:   req.Timezone.Ptr(),
	}
	if v, ok := req.IsDefault.Value(); ok {
		pbReq.IsDefault = v
	}

	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatingCalendarInfo, error) {
			return m.coreClient.CreateOperatingCalendar(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := calendarFromProto(resp)
	return &out, nil
}

func (m *operatingCalendarSvcImpl) UpdateOperatingCalendar(ctx context.Context, req *UpdateOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
	pbReq := &pb.UpdateOperatingCalendarRequest{
		Id:         req.OperatingCalendarID,
		Name:       req.Name.Ptr(),
		DaysOfWeek: req.DaysOfWeek.Ptr(),
		CutoffAt:   field.StringClearableToProto(req.CutoffAt),
		Timezone:   field.StringClearableToProto(req.Timezone),
		IsDefault:  req.IsDefault.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatingCalendarInfo, error) {
			return m.coreClient.UpdateOperatingCalendar(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := calendarFromProto(resp)
	return &out, nil
}

func (m *operatingCalendarSvcImpl) DeleteOperatingCalendar(ctx context.Context, req *DeleteOperatingCalendarRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteOperatingCalendar(ctx, &pb.DeleteOperatingCalendarRequest{Id: req.OperatingCalendarID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (m *operatingCalendarSvcImpl) ListOperatingCalendarClosures(ctx context.Context, req *ListOperatingCalendarClosuresRequest) (*apiresource.List[apiresource.OperatingCalendarClosure], *apierror.APIError) {
	pbReq := &pb.ListOperatingCalendarClosuresRequest{OperatingCalendarId: req.OperatingCalendarID}
	if req.FromDate != nil {
		pbReq.FromDate = timestamppb.New(*req.FromDate)
	}
	if req.ToDate != nil {
		pbReq.ToDate = timestamppb.New(*req.ToDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.list_closures", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOperatingCalendarClosuresResponse, error) {
			return m.coreClient.ListOperatingCalendarClosures(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	closures := make([]apiresource.OperatingCalendarClosure, len(resp.Closures))
	for i, c := range resp.Closures {
		closures[i] = closureFromProto(c)
	}
	return apiresource.NewList(closures, apiresource.PageInfo{}), nil
}

func (m *operatingCalendarSvcImpl) CreateOperatingCalendarClosure(ctx context.Context, req *CreateOperatingCalendarClosureRequest) (*apiresource.OperatingCalendarClosure, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.create_closure", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.OperatingCalendarClosureInfo, error) {
			return m.coreClient.CreateOperatingCalendarClosure(ctx, &pb.CreateOperatingCalendarClosureRequest{
				OperatingCalendarId: req.OperatingCalendarID,
				ClosedOn:            timestamppb.New(req.ClosedOn),
				Name:                req.Name,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := closureFromProto(resp)
	return &out, nil
}

func (m *operatingCalendarSvcImpl) DeleteOperatingCalendarClosure(ctx context.Context, req *DeleteOperatingCalendarClosureRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, operatingCalendarEpSvcTracer, "service.operating_calendars.delete_closure", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteOperatingCalendarClosure(ctx, &pb.DeleteOperatingCalendarClosureRequest{Id: req.ClosureID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}
