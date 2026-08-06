package jobep

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
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

// maps the job's results onto the resource. An unset list stays nil so "no results
// recorded" reads as null, while a present but empty one reads as `[]`.
func jobResultsFromProto(list *pb.JobResultList) []apiresource.JobResult {
	if list == nil {
		return nil
	}
	out := make([]apiresource.JobResult, 0, len(list.GetItems()))
	for _, item := range list.GetItems() {
		out = append(out, apiresource.JobResult{
			Index:          int(item.GetIndex()),
			ID:             item.GetId(),
			Action:         constants.JobResultAction(item.GetAction()),
			SubResourceIDs: item.GetSubResourceIds(),
		})
	}
	return out
}

// maps the job's errors onto the resource, following the same nil/empty rule as
// jobResultsFromProto.
func jobErrorsFromProto(list *pb.JobErrorList) []apierror.RowError {
	if list == nil {
		return nil
	}
	out := make([]apierror.RowError, 0, len(list.GetItems()))
	for _, item := range list.GetItems() {
		entry := apierror.RowError{Error: responseErrorFromJSON(item.GetError())}
		if item.Index != nil {
			idx := int(item.GetIndex())
			entry.Index = &idx
		}
		out = append(out, entry)
	}
	return out
}

// decodes the canonical error object an entry carries. Core-service marshals it from this
// same shared type, so a decode failure is an invariant break rather than bad input.
func responseErrorFromJSON(raw string) apierror.ResponseError {
	if raw == "" {
		return apierror.ResponseError{}
	}
	var e apierror.ResponseError
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		slog.Error("A job error did not decode as the canonical error object", "error", err)
		return apierror.ResponseError{
			Code:    apierror.ErrorCodeInternalError,
			Type:    apierror.ErrorTypeAPI,
			Message: "Something went wrong.",
		}
	}
	return e
}

// maps a JobInfo to the canonical Job resource. Exported because every async operation's
// 202 returns a Job, so every such endpoint reuses this one mapping.
func JobFromProto(info *pb.JobInfo) *apiresource.Job {
	job := &apiresource.Job{
		ID:                info.GetId(),
		Object:            constants.ObjectTypeJob,
		Type:              constants.JobType(info.GetType()),
		Status:            constants.JobStatus(info.GetStatus()),
		CreatedByID:       info.CreatedById,
		CreatedByName:     info.CreatedByName,
		CreatedByUsername: info.CreatedByUsername,
		CreatedByEmail:    info.CreatedByEmail,
		Results:           jobResultsFromProto(info.Results),
		Errors:            jobErrorsFromProto(info.Errors),
		ErrorSummary:      info.ErrorSummary,
		CreatedAt:         info.GetCreatedAt().AsTime(),
		UpdatedAt:         info.GetUpdatedAt().AsTime(),
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
		job.Export = &apiresource.JobExport{URL: url}
	}
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

	return JobFromProto(resp.GetJob()), nil
}
