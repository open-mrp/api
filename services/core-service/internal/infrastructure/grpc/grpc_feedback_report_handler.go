package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegisterFeedbackReportService(server *grpc.Server, feedbackReportSvc domain.FeedbackReportSvc) {
	handler.feedbackReportSvc = feedbackReportSvc
}

func feedbackReportToProto(r *domain.FeedbackReport) *pb.FeedbackReportInfo {
	if r == nil {
		return nil
	}
	return &pb.FeedbackReportInfo{
		Id:        r.ID,
		AccountId: r.AccountID,
		UserId:    r.UserID,
		Message:   r.Message,
		Page:      r.Page,
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

func (h *gRPCHandler) CreateFeedbackReport(ctx context.Context, req *pb.CreateFeedbackReportRequest) (*pb.CreateFeedbackReportResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	report, apiErr := h.feedbackReportSvc.CreateFeedbackReport(ctx, domain.CreateFeedbackReportParams{
		Message: req.Message,
		Page:    req.Page,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateFeedbackReportResponse{
		FeedbackReport: feedbackReportToProto(report),
	}, nil
}

func (h *gRPCHandler) ListFeedbackReports(ctx context.Context, req *pb.ListFeedbackReportsRequest) (*pb.ListFeedbackReportsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.feedbackReportSvc.ListFeedbackReports(ctx, domain.ListFeedbackReportsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	reports := make([]*pb.FeedbackReportInfo, len(result.FeedbackReports))
	for i, r := range result.FeedbackReports {
		reports[i] = feedbackReportToProto(r)
	}

	return &pb.ListFeedbackReportsResponse{
		FeedbackReports: reports,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) BatchGetFeedbackReportsByIDs(ctx context.Context, req *pb.BatchGetFeedbackReportsByIDsRequest) (*pb.BatchGetFeedbackReportsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	reports, apiErr := h.feedbackReportSvc.BatchGetFeedbackReportsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	result := make([]*pb.FeedbackReportInfo, len(reports))
	for i, r := range reports {
		result[i] = feedbackReportToProto(r)
	}

	return &pb.BatchGetFeedbackReportsByIDsResponse{
		FeedbackReports: result,
	}, nil
}
