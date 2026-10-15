package resourceloaders

import (
	"context"

	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

var operatingCalendarLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.operating_calendar")

// LoadOperatingCalendars fetches the calendars records reference, such as an address's receive calendar. IDs the
// caller can't see are left out, so their reference stays null.
func LoadOperatingCalendars(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, operatingCalendarLoaderTracer, "loader.operating_calendars.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOperatingCalendarsResponse, error) {
			return productionScheduleClient.BatchGetOperatingCalendarsByIDs(ctx, &pb.BatchGetOperatingCalendarsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Calendars))
	for _, c := range resp.Calendars {
		out[c.Id] = OperatingCalendarFromProto(c)
	}
	return out, nil
}

// OperatingCalendarFromProto is a calendar as the response shows it.
func OperatingCalendarFromProto(info *pb.OperatingCalendarInfo) *apiresource.OperatingCalendar {
	return &apiresource.OperatingCalendar{
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
