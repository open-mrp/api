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

	params := domain.StartHubspotBackfillParams{}
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

func (h *hubspotSyncGRPCHandler) BulkResolveHubspotCompanyReviews(ctx context.Context, req *pb.BulkResolveHubspotCompanyReviewsRequest) (*pb.BulkResolveHubspotCompanyReviewsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	reviews := make([]domain.ResolveHubspotReviewParams, len(req.Reviews))
	for i, review := range req.Reviews {
		reviews[i] = domain.ResolveHubspotReviewParams{
			ReviewID:          review.ReviewId,
			Action:            review.Action,
			ResolvedHubspotID: review.ResolvedHubspotId,
		}
	}

	job, apiErr := h.svc.BulkResolveReviews(ctx, domain.BulkResolveHubspotReviewsParams{JobID: req.JobId, Reviews: reviews})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BulkResolveHubspotCompanyReviewsResponse{Job: jobToProto(job)}, nil
}

func (h *hubspotSyncGRPCHandler) ExportHubspotCompanyReviews(ctx context.Context, req *pb.ExportHubspotCompanyReviewsRequest) (*pb.ExportHubspotCompanyReviewsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	job, apiErr := h.svc.ExportReviews(ctx, domain.ExportHubspotCompanyReviewsParams{JobID: req.JobId, Status: req.Status})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ExportHubspotCompanyReviewsResponse{Job: jobToProto(job)}, nil
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

func (h *hubspotSyncGRPCHandler) CancelHubspotSync(ctx context.Context, req *pb.CancelHubspotSyncRequest) (*pb.HubspotSyncJobResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	job, apiErr := h.svc.CancelJob(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.HubspotSyncJobResponse{Job: hubspotSyncJobToProto(job)}, nil
}

func (h *hubspotSyncGRPCHandler) ListHubspotSyncRecords(ctx context.Context, req *pb.ListHubspotSyncRecordsRequest) (*pb.ListHubspotSyncRecordsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.svc.ListRecords(ctx, domain.ListHubspotSyncRecordsParams{
		AugnoType: req.AugnoType,
		Cursor:    req.Cursor,
		Limit:     req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	records := make([]*pb.HubspotSyncRecordInfo, 0, len(result.Items))
	for _, record := range result.Items {
		records = append(records, hubspotSyncRecordToProto(record))
	}
	return &pb.ListHubspotSyncRecordsResponse{
		Records: records,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func hubspotSyncRecordToProto(record *domain.HubspotSyncRecord) *pb.HubspotSyncRecordInfo {
	info := &pb.HubspotSyncRecordInfo{
		Id:          record.ID,
		AugnoType:   record.AugnoType,
		AugnoId:     record.AugnoID,
		AugnoName:   record.AugnoName,
		HubspotType: record.HubspotType,
		HubspotId:   record.HubspotID,
		LastError:   record.LastError,
		CreatedAt:   timestamppb.New(record.CreatedAt),
		UpdatedAt:   timestamppb.New(record.UpdatedAt),
	}
	if record.LastSyncedAt != nil {
		info.LastSyncedAt = timestamppb.New(*record.LastSyncedAt)
	}
	return info
}

func hubspotSyncJobToProto(job *domain.HubspotSyncJob) *pb.HubspotSyncJobInfo {
	info := &pb.HubspotSyncJobInfo{
		Id:         job.ID,
		Status:     job.Status,
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
		CustomerEmail:        review.CustomerEmail,
		CustomerUrl:          review.CustomerURL,
		CandidateMatchesJson: string(review.CandidateMatches),
		Status:               review.Status,
		Resolution:           review.Resolution,
		ResolvedHubspotId:    review.ResolvedHubspotID,
		CreatedAt:            timestamppb.New(review.CreatedAt),
		UpdatedAt:            timestamppb.New(review.UpdatedAt),
	}
}
