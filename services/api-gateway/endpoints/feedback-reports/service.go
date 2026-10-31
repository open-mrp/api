package feedbackreportep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type FeedbackReportSvc interface {
	CreateFeedbackReport(ctx context.Context, req *CreateFeedbackReportRequest) (*apiresource.FeedbackReport, *apierror.APIError)
	ListFeedbackReports(ctx context.Context, req *ListFeedbackReportsRequest) (*apiresource.List[apiresource.FeedbackReport], *apierror.APIError)
}

type FeedbackReportSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type feedbackReportSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var feedbackReportSvcTracer = tracing.GetTracer("api-gateway.endpoints.feedback_reports.service")

func (c *FeedbackReportSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("feedback report endpoint service: core client is required")
	}
	return nil
}

func NewFeedbackReportSvc(config *FeedbackReportSvcConfig) FeedbackReportSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &feedbackReportSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *feedbackReportSvcImpl) CreateFeedbackReport(ctx context.Context, req *CreateFeedbackReportRequest) (*apiresource.FeedbackReport, *apierror.APIError) {
	pbReq := &pb.CreateFeedbackReportRequest{
		Message: req.Message,
		Page:    req.Page.Ptr(),
	}
	if severity, ok := req.Severity.Value(); ok {
		s := string(severity)
		pbReq.Severity = &s
	}

	resp, apiErr := grpcutil.CallRPC(ctx, feedbackReportSvcTracer, "service.feedback_reports.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateFeedbackReportResponse, error) {
			return m.coreClient.CreateFeedbackReport(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return resourceloaders.FeedbackReportFromProto(ctx, resp.FeedbackReport), nil
}

func (m *feedbackReportSvcImpl) ListFeedbackReports(ctx context.Context, req *ListFeedbackReportsRequest) (*apiresource.List[apiresource.FeedbackReport], *apierror.APIError) {
	pbReq := &pb.ListFeedbackReportsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, feedbackReportSvcTracer, "service.feedback_reports.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListFeedbackReportsResponse, error) {
			return m.coreClient.ListFeedbackReports(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	reports := make([]apiresource.FeedbackReport, 0, len(resp.FeedbackReports))
	for _, r := range resp.FeedbackReports {
		reports = append(reports, *resourceloaders.FeedbackReportFromProto(ctx, r))
	}

	return apiresource.NewList(reports, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}
