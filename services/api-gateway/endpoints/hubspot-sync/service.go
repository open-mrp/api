package hubspotsyncep

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	jobep "github.com/augno/api/services/api-gateway/endpoints/jobs"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HubspotSyncSvc backs the HubSpot backfill endpoints by calling the core-service gRPC API.
type HubspotSyncSvc interface {
	StartSync(ctx context.Context, req *StartHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError)
	GetCurrentSyncJob(ctx context.Context, req *GetCurrentHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError)
	GetSyncJob(ctx context.Context, req *GetHubspotSyncJobRequest) (*apiresource.HubspotSyncJob, *apierror.APIError)
	CancelSync(ctx context.Context, req *CancelHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError)
	ListSyncRecords(ctx context.Context, req *ListHubspotSyncRecordsRequest) (*apiresource.List[apiresource.HubspotSyncRecord], *apierror.APIError)
	ListCompanyReviews(ctx context.Context, req *ListHubspotCompanyReviewsRequest) (*apiresource.List[apiresource.HubspotCompanyReview], *apierror.APIError)
	LinkCompanyReview(ctx context.Context, req *LinkHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError)
	CreateNewCompanyReview(ctx context.Context, req *CreateNewHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError)
	SkipCompanyReview(ctx context.Context, req *SkipHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError)
	BulkResolveCompanyReviews(ctx context.Context, req *BulkResolveHubspotCompanyReviewsRequest) (*apiresource.Job, *apierror.APIError)
	ExportCompanyReviews(ctx context.Context, req *ExportHubspotCompanyReviewsRequest) (*apiresource.Job, *apierror.APIError)
	ExecuteSync(ctx context.Context, req *ExecuteHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError)
}

// HubspotSyncSvcConfig configures the HubSpot sync endpoint service.
type HubspotSyncSvcConfig struct {
	// CoreClient (required) is the core-service HubSpot sync gRPC client.
	CoreClient pb.CoreHubspotSyncServiceClient
}

type hubspotSyncSvcImpl struct {
	coreClient pb.CoreHubspotSyncServiceClient
}

var hubspotSyncSvcTracer = tracing.GetTracer("api-gateway.endpoints.hubspot-sync.service")

func (c *HubspotSyncSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("hubspot sync endpoint service: core client is required")
	}
	return nil
}

func NewHubspotSyncSvc(config *HubspotSyncSvcConfig) HubspotSyncSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &hubspotSyncSvcImpl{coreClient: config.CoreClient}
}

func (m *hubspotSyncSvcImpl) StartSync(ctx context.Context, req *StartHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
	pbReq := &pb.StartHubspotBackfillRequest{}
	if cutoff, ok := req.GoLiveCutoffAt.Value(); ok {
		pbReq.GoliveCutoffAt = timestamppb.New(cutoff)
	}
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.start", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.HubspotSyncJobResponse, error) {
			return m.coreClient.StartHubspotBackfill(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return hubspotSyncJobFromProto(resp.Job), nil
}

func (m *hubspotSyncSvcImpl) GetCurrentSyncJob(ctx context.Context, _ *GetCurrentHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.get_current", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.HubspotSyncJobResponse, error) {
			return m.coreClient.GetCurrentHubspotSyncJob(ctx, &pb.GetCurrentHubspotSyncJobRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return hubspotSyncJobFromProto(resp.Job), nil
}

func (m *hubspotSyncSvcImpl) GetSyncJob(ctx context.Context, req *GetHubspotSyncJobRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.HubspotSyncJobResponse, error) {
			return m.coreClient.GetHubspotSyncJob(ctx, &pb.GetHubspotSyncJobRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return hubspotSyncJobFromProto(resp.Job), nil
}

func (m *hubspotSyncSvcImpl) CancelSync(ctx context.Context, req *CancelHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.cancel", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.HubspotSyncJobResponse, error) {
			return m.coreClient.CancelHubspotSync(ctx, &pb.CancelHubspotSyncRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return hubspotSyncJobFromProto(resp.Job), nil
}

func (m *hubspotSyncSvcImpl) ListSyncRecords(ctx context.Context, req *ListHubspotSyncRecordsRequest) (*apiresource.List[apiresource.HubspotSyncRecord], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.list_records", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListHubspotSyncRecordsResponse, error) {
			return m.coreClient.ListHubspotSyncRecords(ctx, &pb.ListHubspotSyncRecordsRequest{
				AugnoType: string(req.AugnoType),
				Cursor:    req.Cursor,
				Limit:     req.Limit,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.HubspotSyncRecord, 0, len(resp.Records))
	for _, record := range resp.Records {
		items = append(items, *hubspotSyncRecordFromProto(record))
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *hubspotSyncSvcImpl) ListCompanyReviews(ctx context.Context, req *ListHubspotCompanyReviewsRequest) (*apiresource.List[apiresource.HubspotCompanyReview], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.list_reviews", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListHubspotCompanyReviewsResponse, error) {
			return m.coreClient.ListHubspotCompanyReviews(ctx, &pb.ListHubspotCompanyReviewsRequest{JobId: req.JobID, Status: req.Status.StringPtr()}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.HubspotCompanyReview, 0, len(resp.Reviews))
	for _, review := range resp.Reviews {
		items = append(items, *hubspotCompanyReviewFromProto(review))
	}
	// The review queue for a job is bounded and returned in full, so there is no pagination.
	return apiresource.NewList(items, apiresource.PageInfo{}), nil
}

func (m *hubspotSyncSvcImpl) LinkCompanyReview(ctx context.Context, req *LinkHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
	return m.resolveCompanyReview(ctx, req.JobID, req.ReviewID, "link", &req.ResolvedHubspotID)
}

func (m *hubspotSyncSvcImpl) CreateNewCompanyReview(ctx context.Context, req *CreateNewHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
	return m.resolveCompanyReview(ctx, req.JobID, req.ReviewID, "create_new", nil)
}

func (m *hubspotSyncSvcImpl) SkipCompanyReview(ctx context.Context, req *SkipHubspotCompanyReviewRequest) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
	return m.resolveCompanyReview(ctx, req.JobID, req.ReviewID, "skip", nil)
}

// resolveCompanyReview applies a resolution action to a company review via the core gRPC API.
func (m *hubspotSyncSvcImpl) resolveCompanyReview(ctx context.Context, jobID, reviewID, action string, resolvedHubspotID *string) (*apiresource.HubspotCompanyReview, *apierror.APIError) {
	pbReq := &pb.ResolveHubspotCompanyReviewRequest{
		JobId:             jobID,
		ReviewId:          reviewID,
		Action:            action,
		ResolvedHubspotId: resolvedHubspotID,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.resolve_review", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.HubspotCompanyReviewResponse, error) {
			return m.coreClient.ResolveHubspotCompanyReview(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return hubspotCompanyReviewFromProto(resp.Review), nil
}

func (m *hubspotSyncSvcImpl) BulkResolveCompanyReviews(ctx context.Context, req *BulkResolveHubspotCompanyReviewsRequest) (*apiresource.Job, *apierror.APIError) {
	reviews := make([]*pb.HubspotCompanyReviewResolution, len(req.Reviews))
	for i, review := range req.Reviews {
		reviews[i] = &pb.HubspotCompanyReviewResolution{
			ReviewId:          review.ReviewID,
			Action:            string(review.Action),
			ResolvedHubspotId: review.ResolvedHubspotID,
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.bulk_resolve_reviews", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkResolveHubspotCompanyReviewsResponse, error) {
			return m.coreClient.BulkResolveHubspotCompanyReviews(ctx, &pb.BulkResolveHubspotCompanyReviewsRequest{JobId: req.JobID, Reviews: reviews}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *hubspotSyncSvcImpl) ExportCompanyReviews(ctx context.Context, req *ExportHubspotCompanyReviewsRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.export_reviews", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportHubspotCompanyReviewsResponse, error) {
			return m.coreClient.ExportHubspotCompanyReviews(ctx, &pb.ExportHubspotCompanyReviewsRequest{JobId: req.JobID, Status: req.Status.StringPtr()}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *hubspotSyncSvcImpl) ExecuteSync(ctx context.Context, req *ExecuteHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, hubspotSyncSvcTracer, "service.hubspot-sync.execute", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.HubspotSyncJobResponse, error) {
			return m.coreClient.ExecuteHubspotSync(ctx, &pb.ExecuteHubspotSyncRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return hubspotSyncJobFromProto(resp.Job), nil
}

func hubspotSyncJobFromProto(p *pb.HubspotSyncJobInfo) *apiresource.HubspotSyncJob {
	job := &apiresource.HubspotSyncJob{
		ID:             p.Id,
		Object:         constants.ObjectTypeHubspotSyncJob,
		Status:         constants.HubspotSyncJobStatus(p.Status),
		GoLiveCutoffAt: timePtr(p.GoliveCutoffAt),
		LastError:      p.LastError,
		StartedAt:      timePtr(p.StartedAt),
		CompletedAt:    timePtr(p.CompletedAt),
		CreatedAt:      grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(p.UpdatedAt),
	}
	if p.CountsJson != "" {
		var report apiresource.HubspotSyncReport
		if err := json.Unmarshal([]byte(p.CountsJson), &report); err == nil {
			report.Object = constants.ObjectTypeHubspotSyncReport
			job.Report = &report
		}
	}
	return job
}

func hubspotSyncRecordFromProto(p *pb.HubspotSyncRecordInfo) *apiresource.HubspotSyncRecord {
	return &apiresource.HubspotSyncRecord{
		ID:           p.Id,
		Object:       constants.ObjectTypeHubspotSyncRecord,
		AugnoType:    constants.HubspotSyncRecordAugnoType(p.AugnoType),
		AugnoID:      p.AugnoId,
		AugnoName:    p.AugnoName,
		HubspotType:  constants.HubspotSyncRecordHubspotType(p.HubspotType),
		HubspotID:    p.HubspotId,
		LastSyncedAt: timePtr(p.LastSyncedAt),
		LastError:    p.LastError,
		CreatedAt:    grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func hubspotCompanyReviewFromProto(p *pb.HubspotCompanyReviewInfo) *apiresource.HubspotCompanyReview {
	candidates := []apiresource.HubspotCompanyCandidate{}
	if p.CandidateMatchesJson != "" {
		_ = json.Unmarshal([]byte(p.CandidateMatchesJson), &candidates)
	}
	for i := range candidates {
		candidates[i].Object = constants.ObjectTypeHubspotCompanyCandidate
	}
	return &apiresource.HubspotCompanyReview{
		ID:     p.Id,
		Object: constants.ObjectTypeHubspotCompanyReview,
		Job:               &apiresource.HubspotSyncJob{ID: p.JobId, Object: constants.ObjectTypeHubspotSyncJob},
		Customer:          &apiresource.Customer{ID: p.AugnoCustomerId, Object: constants.ObjectTypeCustomer, Name: p.CustomerName},
		CustomerEmail:     p.CustomerEmail,
		CustomerURL:       p.CustomerUrl,
		Candidates:        apiresource.NewList(candidates, apiresource.PageInfo{}),
		Status:            constants.HubspotCompanyReviewStatus(p.Status),
		Resolution:        p.Resolution,
		ResolvedHubspotID: p.ResolvedHubspotId,
		CreatedAt:         grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
