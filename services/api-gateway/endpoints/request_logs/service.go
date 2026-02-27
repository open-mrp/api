package requestlogep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RequestLogSvc interface {
	ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLogListItem], *apierror.APIError)
	GetRequestLog(ctx context.Context, req *GetRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError)
}

type RequestLogSvcConfig struct {
	LoggingClient pb.LoggingServiceClient
}

type requestLogSvcImpl struct {
	loggingClient pb.LoggingServiceClient
}

var requestLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.request_logs.service")

func (c *RequestLogSvcConfig) validate() error {
	if c.LoggingClient == nil {
		return fmt.Errorf("request log endpoint service: logging client is required")
	}
	return nil
}

func NewRequestLogSvc(config *RequestLogSvcConfig) RequestLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &requestLogSvcImpl{
		loggingClient: config.LoggingClient,
	}
}

func requireInternalAdmin(ctx context.Context) *apierror.APIError {
	identity, apiErr := httptransport.GetIdentity(ctx)
	if apiErr != nil {
		return apiErr
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return apierror.NewAuthorizationError("Only internal administrators can access request logs.")
	}
	return nil
}

func (m *requestLogSvcImpl) ListRequestLogs(ctx context.Context, req *ListRequestLogsRequest) (*apiresource.List[apiresource.RequestLogListItem], *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.ListRequestLogsRequest{
		Query:      req.Query,
		Method:     req.Method,
		ErrorCode:  req.ErrorCode,
		AccountId:  req.AccountID,
		ActorId:    req.ActorID,
		ActorType:  req.ActorType,
		ActorName:  req.ActorName,
		ExactMatch: req.ExactMatch,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
	}

	if req.StartDate != nil && !req.StartDate.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}
	if req.StatusCode != nil {
		pbReq.StatusCode = req.StatusCode
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListRequestLogsResponse, error) {
			return m.loggingClient.ListRequestLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return RequestLogListPresenter(resp), nil
}

func (m *requestLogSvcImpl) GetRequestLog(ctx context.Context, req *GetRequestLogRequest) (*apiresource.RequestLog, *apierror.APIError) {
	if apiErr := requireInternalAdmin(ctx); apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.GetRequestLogRequest{
		Id: req.ID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, requestLogSvcTracer, "service.request_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRequestLogResponse, error) {
			return m.loggingClient.GetRequestLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := RequestLogPresenter(resp.RequestLog)
	return &result, nil
}
