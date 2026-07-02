package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type hubspotSyncGRPCHandler struct {
	pb.UnimplementedCoreHubspotSyncServiceServer
	svc domain.HubspotSyncSvc
}

func RegisterHubspotSyncService(server *grpc.Server, svc domain.HubspotSyncSvc) {
	pb.RegisterCoreHubspotSyncServiceServer(server, &hubspotSyncGRPCHandler{svc: svc})
}

func (h *hubspotSyncGRPCHandler) StartHubspotBackfill(ctx context.Context, req *pb.StartHubspotBackfillRequest) (*pb.HubspotSyncJobResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.StartHubspotBackfillParams{DryRun: req.DryRun}
	if req.GoliveCutoffAt != nil {
		t := req.GoliveCutoffAt.AsTime()
		params.GoLiveCutoffAt = &t
	}
	job, apiErr := h.svc.StartBackfill(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.HubspotSyncJobResponse{Job: hubspotSyncJobToProto(job)}, nil
}

func (h *hubspotSyncGRPCHandler) GetCurrentHubspotSyncJob(ctx context.Context, req *pb.GetCurrentHubspotSyncJobRequest) (*pb.HubspotSyncJobResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	job, apiErr := h.svc.GetCurrentJob(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.HubspotSyncJobResponse{Job: hubspotSyncJobToProto(job)}, nil
}

func (h *hubspotSyncGRPCHandler) GetHubspotSyncJob(ctx context.Context, req *pb.GetHubspotSyncJobRequest) (*pb.HubspotSyncJobResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	job, apiErr := h.svc.GetJob(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.HubspotSyncJobResponse{Job: hubspotSyncJobToProto(job)}, nil
}

func (h *hubspotSyncGRPCHandler) ListHubspotCompanyReviews(ctx context.Context, req *pb.ListHubspotCompanyReviewsRequest) (*pb.ListHubspotCompanyReviewsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	reviews, apiErr := h.svc.ListReviews(ctx, req.JobId, req.Status)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbReviews := make([]*pb.HubspotCompanyReviewInfo, len(reviews))
	for i, review := range reviews {
		pbReviews[i] = hubspotCompanyReviewToProto(review)
	}
	return &pb.ListHubspotCompanyReviewsResponse{Reviews: pbReviews}, nil
}

func (h *hubspotSyncGRPCHandler) ResolveHubspotCompanyReview(ctx context.Context, req *pb.ResolveHubspotCompanyReviewRequest) (*pb.HubspotCompanyReviewResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	review, apiErr := h.svc.ResolveReview(ctx, domain.ResolveHubspotReviewParams{
		ReviewID:          req.ReviewId,
		Action:            req.Action,
		ResolvedHubspotID: req.ResolvedHubspotId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.HubspotCompanyReviewResponse{Review: hubspotCompanyReviewToProto(review)}, nil
}

func (h *hubspotSyncGRPCHandler) ExecuteHubspotSync(ctx context.Context, req *pb.ExecuteHubspotSyncRequest) (*pb.HubspotSyncJobResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	job, apiErr := h.svc.StartExecute(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.HubspotSyncJobResponse{Job: hubspotSyncJobToProto(job)}, nil
}

func hubspotSyncJobToProto(job *domain.HubspotSyncJob) *pb.HubspotSyncJobInfo {
	info := &pb.HubspotSyncJobInfo{
		Id:         job.ID,
		Status:     job.Status,
		DryRun:     job.DryRun,
		CountsJson: string(job.Counts),
		LastError:  job.LastError,
		CreatedAt:  timestamppb.New(job.CreatedAt),
		UpdatedAt:  timestamppb.New(job.UpdatedAt),
	}
	if job.GoLiveCutoffAt != nil {
		info.GoliveCutoffAt = timestamppb.New(*job.GoLiveCutoffAt)
	}
	if job.StartedAt != nil {
		info.StartedAt = timestamppb.New(*job.StartedAt)
	}
	if job.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*job.CompletedAt)
	}
	return info
}

func hubspotCompanyReviewToProto(review *domain.HubspotCompanyReview) *pb.HubspotCompanyReviewInfo {
	return &pb.HubspotCompanyReviewInfo{
		Id:                   review.ID,
		JobId:                review.JobID,
		AugnoCustomerId:      review.AugnoCustomerID,
		CustomerName:         review.CustomerName,
		CandidateMatchesJson: string(review.CandidateMatches),
		Status:               review.Status,
		Resolution:           review.Resolution,
		ResolvedHubspotId:    review.ResolvedHubspotID,
		CreatedAt:            timestamppb.New(review.CreatedAt),
		UpdatedAt:            timestamppb.New(review.UpdatedAt),
	}
}
