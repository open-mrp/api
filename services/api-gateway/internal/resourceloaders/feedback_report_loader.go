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

var feedbackReportLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.feedback_report")

func LoadFeedbackReports(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, feedbackReportLoaderTracer, "loader.feedback_reports.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetFeedbackReportsByIDsResponse, error) {
			return coreClient.BatchGetFeedbackReportsByIDs(ctx, &pb.BatchGetFeedbackReportsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.FeedbackReports))
	for _, r := range resp.FeedbackReports {
		out[r.Id] = FeedbackReportFromProto(ctx, r)
	}
	return out, nil
}

func FeedbackReportFromProto(ctx context.Context, r *pb.FeedbackReportInfo) *apiresource.FeedbackReport {
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeFeedbackReport, r.Id, "user_id", r.UserId)
	return &apiresource.FeedbackReport{
		ID:        r.Id,
		Object:    constants.ObjectTypeFeedbackReport,
		Message:   r.Message,
		Page:      r.Page,
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}
}
