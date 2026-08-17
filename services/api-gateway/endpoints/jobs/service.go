package jobep

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
)

var jobEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.jobs.service")

type JobSvc interface {
	GetJob(ctx context.Context, req *RetrieveJobRequest) (*apiresource.Job, *apierror.APIError)
	CancelJob(ctx context.Context, req *CancelJobRequest) (*apiresource.Job, *apierror.APIError)
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
// A trimmed record is reported as has_next_page: the job carries fewer rows than it
// produced, and saying so is the only honest thing a bounded list can do — there is no
// cursor to follow, because the rows it dropped were never stored.
func jobResultsFromProto(list *pb.JobResultList) *apiresource.List[apiresource.JobResult] {
	if list == nil {
		return nil
	}
	out := make([]apiresource.JobResult, 0, len(list.GetItems()))
	for _, item := range list.GetItems() {
		result := apiresource.JobResult{
			Object: constants.ObjectTypeJobResult,
			Index:  int(item.GetIndex()),
			Status: constants.JobResultStatus(item.GetStatus()),
		}
		if id := item.GetId(); id != "" {
			result.Resource = apiresource.NewEntity(id, constants.ObjectType(item.GetResourceType()), nil, nil)
		}
		if subs := item.GetSubResources(); len(subs) > 0 {
			entities := make([]apiresource.Entity, 0, len(subs))
			for _, sub := range subs {
				entities = append(entities, *apiresource.NewEntity(sub.GetId(), constants.ObjectType(sub.GetResourceType()), nil, nil))
			}
			result.SubResources = apiresource.NewList(entities, apiresource.PageInfo{})
		}
		if raw := item.GetError(); raw != "" {
			result.Error = responseErrorFromJSON(raw)
		}
		out = append(out, result)
	}
	return apiresource.NewList(out, apiresource.PageInfo{HasNextPage: list.GetTruncated()})
}

// decodes the canonical error object an entry carries. Core-service marshals it from this
// same shared type, so a decode failure is an invariant break rather than bad input.
func responseErrorFromJSON(raw string) *apierror.ResponseError {
	if raw == "" {
		return nil
	}
	var e apierror.ResponseError
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		slog.Error("A job error did not decode as the canonical error object", "error", err)
		return &apierror.ResponseError{
			Code:    apierror.ErrorCodeInternalError,
			Type:    apierror.ErrorTypeAPI,
			Message: "Something went wrong.",
		}
	}
	return &e
}

// maps a JobInfo to the canonical Job resource. Exported because every async operation's
// 202 returns a Job, so every such endpoint reuses this one mapping.
//
// created_by is left nil: it is expandable, and the creator's id is stashed for the
// include resolver by StashJobMeta rather than inlined here.
func JobFromProto(info *pb.JobInfo) *apiresource.Job {
	job := &apiresource.Job{
		ID:        info.GetId(),
		Object:    constants.ObjectTypeJob,
		Type:      constants.JobType(info.GetType()),
		Status:    constants.JobStatus(info.GetStatus()),
		Results:   jobResultsFromProto(info.Results),
		Error:     responseErrorFromJSON(info.GetError()),
		CreatedAt: info.GetCreatedAt().AsTime(),
		UpdatedAt: info.GetUpdatedAt().AsTime(),
	}

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
		job.CancelledAt = &t
	}

	return job
}

// StashJobMeta records the creator's id for the created_by include and preheats the actor
// built from it — LoadActors has no backing store. The actor is returned (nil when the job
// has no creator) so the caller can batch-hydrate display names.
func StashJobMeta(ctx context.Context, info *pb.JobInfo, jobID string) *apiresource.Actor {
	actor := resourceloaders.ActorRefFromID(info.GetCreatedById())
	if actor == nil {
		return nil
	}
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeJob, jobID, "created_by_id", actor.ID)
	resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
	return actor
}

// hydrateCreator fills the creator's display name + handle. A no-op unless the caller
// expanded created_by — the only case where the name is rendered — avoiding a needless
// loader round-trip on the 202 of every async operation.
func hydrateCreator(ctx context.Context, creator *apiresource.Actor) {
	if creator == nil || !resourcekit.RequestedIncludeSet(ctx)["created_by"] {
		return
	}
	resourceloaders.HydrateIdentityActorNames(ctx, []*apiresource.Actor{creator})
}

// reads a job. A completed export carries the link to its file alongside the rest of it.
func (m *jobSvcImpl) GetJob(ctx context.Context, req *RetrieveJobRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, jobEpSvcTracer, "service.jobs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetJobResponse, error) {
			return m.coreClient.GetJob(ctx, &pb.GetJobRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	job := JobFromProto(resp.GetJob())
	if url := resp.GetJob().GetExport().GetUrl(); url != "" {
		job.Export = apiresource.NewJobExport(url)
	}
	hydrateCreator(ctx, StashJobMeta(ctx, resp.GetJob(), job.ID))
	return job, nil
}

func (m *jobSvcImpl) CancelJob(ctx context.Context, req *CancelJobRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, jobEpSvcTracer, "service.jobs.cancel", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CancelJobResponse, error) {
			return m.coreClient.CancelJob(ctx, &pb.CancelJobRequest{Id: req.JobID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	job := JobFromProto(resp.GetJob())
	hydrateCreator(ctx, StashJobMeta(ctx, resp.GetJob(), job.ID))
	return job, nil
}
