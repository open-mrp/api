package emaillogep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type EmailLogSvc interface {
	ListEmailLogs(ctx context.Context, req *ListEmailLogsRequest) (*apiresource.List[apiresource.EmailLog], *apierror.APIError)
	GetEmailLog(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError)
}

type EmailLogSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type emailLogSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var emailLogSvcTracer = tracing.GetTracer("api-gateway.endpoints.email_logs.service")

func (c *EmailLogSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("email log endpoint service: core client is required")
	}
	return nil
}

func NewEmailLogSvc(config *EmailLogSvcConfig) EmailLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &emailLogSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *emailLogSvcImpl) ListEmailLogs(ctx context.Context, req *ListEmailLogsRequest) (*apiresource.List[apiresource.EmailLog], *apierror.APIError) {
	pbReq := &pb.ListEmailLogsRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, emailLogSvcTracer, "service.email_logs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListEmailLogsResponse, error) {
			return m.coreClient.ListEmailLogs(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return EmailLogListPresenter(ctx, resp), nil
}

func (m *emailLogSvcImpl) GetEmailLog(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError) {
	pbReq := &pb.GetEmailLogRequest{
		Id:       req.EmailLogID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, emailLogSvcTracer, "service.email_logs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetEmailLogResponse, error) {
			return m.coreClient.GetEmailLog(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := EmailLogPresenter(resp.EmailLog)
	return &result, nil
}
