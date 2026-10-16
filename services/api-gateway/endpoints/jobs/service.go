package jobep

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"

	"google.golang.org/grpc"
)

var jobEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.jobs.service")

type JobSvc interface {
	GetJob(ctx context.Context, req *RetrieveJobRequest) (*apiresource.AsyncJob, *apierror.APIError)
	ListJobs(ctx context.Context, req *ListJobsRequest) (*apiresource.List[apiresource.AsyncJob], *apierror.APIError)
	CancelJob(ctx context.Context, req *CancelJobRequest) (*apiresource.AsyncJob, *apierror.APIError)
}

type jobSvcImpl struct {
	coreClient pb.CoreJobServiceClient
}

type JobSvcConfig struct {
	// CoreClient (required) is the core-service job gRPC client.
	CoreClient pb.CoreJobServiceClient
}

func (c *JobSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("job endpoint service: core client is required")
	}
	return nil
}

func NewJobSvc(config *JobSvcConfig) JobSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &jobSvcImpl{coreClient: config.CoreClient}
}

// maps the job's row outcomes onto the resource. An unset list stays nil so "no results
// recorded" reads as null, while a present but empty one reads as an empty list.
//
// A trimmed record has no next page to point at: the rows it dropped were never stored.
func jobResultsFromProto(list *pb.JobResultList) *apiresource.List[apiresource.AsyncJobResult] {
	if list == nil {
		return nil
	}
	out := make([]apiresource.AsyncJobResult, 0, len(list.GetItems()))
	for _, item := range list.GetItems() {
		result := apiresource.AsyncJobResult{
			Object: object.TypeAsyncJobResult,
			Index:  int(item.GetIndex()),
			Status: object.AsyncJobResultStatus(item.GetStatus()),
		}
		if id := item.GetId(); id != "" {
			result.Resource = apiresource.NewEntity(id, constants.ObjectType(item.GetResourceType()), item.Name, nil)
		}
		if subs := item.GetSubResources(); len(subs) > 0 {
			entities := make([]apiresource.Entity, 0, len(subs))
			for _, sub := range subs {
				entities = append(entities, *apiresource.NewEntity(sub.GetId(), constants.ObjectType(sub.GetResourceType()), sub.Name, nil))
			}
			result.SubResources = apiresource.NewList(entities, apiresource.PageInfo{})
		}
		if raw := item.GetError(); raw != "" {
			result.Error = responseErrorFromJSON(raw)
		}
		out = append(out, result)
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}

// decodes the canonical error object an entry carries. Core-service marshals it from this
// same shared type, so a decode failure is an invariant break rather than bad input.
func responseErrorFromJSON(raw string) *apierror.ErrorObject {
	if raw == "" {
		return nil
	}
	var e apierror.ErrorObject
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		slog.Error("A job error did not decode as the canonical error object", "error", err)
		return &apierror.ErrorObject{
			Code:    apierror.CodeInternalError,
			Type:    apierror.TypeAPI,
			Message: "Something went wrong.",
		}
	}
	return &e
}

// JobFromProto maps a JobInfo to the async_job resource, with its creator's name resolved. Exported because every
// async operation's 202 returns one, so every such endpoint reuses this one mapping.
func JobFromProto(ctx context.Context, info *pb.JobInfo) *apiresource.AsyncJob {
	job := jobFromProto(info)
	resourceloaders.ResolveActorRefs(ctx, []*apiresource.ActorRef{job.CreatedBy})
	return job
}

// jobFromProto maps a JobInfo without resolving its creator's name, so a list can resolve every creator at once.
func jobFromProto(info *pb.JobInfo) *apiresource.AsyncJob {
	job := &apiresource.AsyncJob{AsyncJob: object.AsyncJob[apiresource.Entity]{
		ID:        info.GetId(),
		Object:    object.TypeAsyncJob,
		Type:      info.GetType(),
		Status:    object.AsyncJobStatus(info.GetStatus()),
		Results:   jobResultsFromProto(info.Results),
		Error:     responseErrorFromJSON(info.GetError()),
		CreatedAt: info.GetCreatedAt().AsTime(),
		UpdatedAt: info.GetUpdatedAt().AsTime(),
	}, CreatedBy: jobCreator(info.GetCreatedById())}

	if resourceType := info.GetResourceType(); resourceType != "" {
		t := constants.ObjectType(resourceType)
		job.ResourceType = &t
	}
	if info.StartedAt != nil {
		t := info.GetStartedAt().AsTime()
		job.StartedAt = &t
	}
	if info.CompletedAt != nil {
		t := info.GetCompletedAt().AsTime()
		job.CompletedAt = &t
	}
	if info.FailedAt != nil {
		t := info.GetFailedAt().AsTime()
		job.FailedAt = &t
	}
	if info.CancelledAt != nil {
		t := info.GetCancelledAt().AsTime()
		job.CanceledAt = &t
	}
	if url := info.GetExport().GetUrl(); url != "" {
		job.Export = apiresource.NewAsyncJobExport(url)
	}

	return job
}

// jobCreator is the job's creator. A job raised with no recorded creator was raised by the platform itself.
func jobCreator(createdByID string) *apiresource.ActorRef {
	if ref := resourceloaders.ActorRefFromBareID(createdByID); ref != nil {
		return ref
	}
	return resourceloaders.ActorRefFromRecorded(constants.ActorTypeSystem, "", nil)
}

// reads a job. A completed export carries the link to its file alongside the rest of it.
func (m *jobSvcImpl) GetJob(ctx context.Context, req *RetrieveJobRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, jobEpSvcTracer, "service.jobs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetJobResponse, error) {
			return m.coreClient.GetJob(ctx, &pb.GetJobRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return JobFromProto(ctx, resp.GetJob()), nil
}

func (m *jobSvcImpl) ListJobs(ctx context.Context, req *ListJobsRequest) (*apiresource.List[apiresource.AsyncJob], *apierror.APIError) {
	pbReq := &pb.ListJobsRequest{Cursor: req.Cursor, Limit: req.Limit}
	for _, t := range req.Types {
		pbReq.Types = append(pbReq.Types, string(t))
	}
	for _, s := range req.Statuses {
		pbReq.Statuses = append(pbReq.Statuses, string(s))
	}

	resp, apiErr := grpcutil.CallRPC(ctx, jobEpSvcTracer, "service.jobs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListJobsResponse, error) {
			return m.coreClient.ListJobs(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	jobs := make([]apiresource.AsyncJob, len(resp.GetJobs()))
	creators := make([]*apiresource.ActorRef, len(resp.GetJobs()))
	for i, info := range resp.GetJobs() {
		jobs[i] = *jobFromProto(info)
		creators[i] = jobs[i].CreatedBy
	}
	resourceloaders.ResolveActorRefs(ctx, creators)

	return apiresource.NewList(jobs, grpcutil.MapProtoPageInfo(ctx, resp.GetPageInfo())), nil
}

func (m *jobSvcImpl) CancelJob(ctx context.Context, req *CancelJobRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, jobEpSvcTracer, "service.jobs.cancel", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CancelJobResponse, error) {
			return m.coreClient.CancelJob(ctx, &pb.CancelJobRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return JobFromProto(ctx, resp.GetJob()), nil
}
