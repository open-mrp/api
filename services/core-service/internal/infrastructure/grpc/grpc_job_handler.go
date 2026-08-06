package grpc

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type jobGRPCHandler struct {
	pb.UnimplementedCoreJobServiceServer

	jobSvc    domain.JobSvc
	exportSvc domain.ExportSvc
}

func RegisterJobService(server *grpc.Server, jobSvc domain.JobSvc, exportSvc domain.ExportSvc) {
	pb.RegisterCoreJobServiceServer(server, &jobGRPCHandler{jobSvc: jobSvc, exportSvc: exportSvc})
}

// jobResultsToProto maps the job's results onto the wire. A nil list stays unset —
// the job has recorded no results — while an empty one is sent as a present, empty
// list, which is how "ran and wrote nothing" survives the crossing.
func jobResultsToProto(results []domain.RowResult) *pb.JobResultList {
	if results == nil {
		return nil
	}
	items := make([]*pb.JobResultInfo, len(results))
	for i, r := range results {
		items[i] = &pb.JobResultInfo{
			Index:          int32(r.Index),
			Id:             r.ID,
			Action:         string(r.Action),
			SubResourceIds: r.SubResourceIDs,
		}
	}
	return &pb.JobResultList{Items: items}
}

// jobErrorsToProto maps the job's errors onto the wire, following the same nil/empty
// rule as jobResultsToProto.
func jobErrorsToProto(errs []apierror.RowError) *pb.JobErrorList {
	if errs == nil {
		return nil
	}
	items := make([]*pb.JobErrorInfo, len(errs))
	for i, e := range errs {
		item := &pb.JobErrorInfo{Error: marshalResponseError(e.Error)}
		if e.Index != nil {
			idx := int32(*e.Index)
			item.Index = &idx
		}
		items[i] = item
	}
	return &pb.JobErrorList{Items: items}
}

// marshalResponseError renders the canonical client-facing error object for the wire.
// It stays JSON rather than becoming proto fields because the gateway unmarshals it
// straight back into this same shared type, so there is no second definition to keep in
// step. ResponseError is plain data, so this cannot fail in practice; an entry that says
// nothing would be worse than one that says something, hence the fallback.
func marshalResponseError(e apierror.ResponseError) string {
	raw, err := json.Marshal(e)
	if err != nil {
		slog.Error("Failed to encode a job error for the wire", "error", err, "code", e.Code)
		return `{"code":"internal_error","type":"api_error","message":"Something went wrong."}`
	}
	return string(raw)
}

func optionalTimestampToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func jobToProto(job *domain.Job) *pb.JobInfo {
	return &pb.JobInfo{
		Id:     job.ID,
		Type:   string(job.Type),
		Status: string(job.Status()),

		CreatedById:       job.CreatedByID,
		CreatedByName:     job.CreatedByName,
		CreatedByUsername: job.CreatedByUsername,
		CreatedByEmail:    job.CreatedByEmail,

		Results:      jobResultsToProto(job.Results),
		Errors:       jobErrorsToProto(job.Errors),
		ErrorSummary: job.ErrorSummary,

		StartedAt:   optionalTimestampToProto(job.StartedAt),
		CompletedAt: optionalTimestampToProto(job.CompletedAt),
		FailedAt:    optionalTimestampToProto(job.FailedAt),
		CancelledAt: optionalTimestampToProto(job.CancelledAt),
		CreatedAt:   timestamppb.New(job.CreatedAt),
		UpdatedAt:   timestamppb.New(job.UpdatedAt),
	}
}

func (h *jobGRPCHandler) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	job, apiErr := h.jobSvc.GetJob(ctx, req.GetId())
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	info := jobToProto(job)

	// Signed here rather than stored: a link costs no I/O, and one that expires cannot be
	// stale on a job read back a day later.
	url, apiErr := h.exportSvc.DownloadURL(ctx, job)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	if url != "" {
		info.Export = &pb.ExportInfo{Url: url}
	}

	return &pb.GetJobResponse{Job: info}, nil
}

func (h *jobGRPCHandler) CancelJob(ctx context.Context, req *pb.CancelJobRequest) (*pb.CancelJobResponse, error) {
	job, apiErr := h.jobSvc.CancelJob(ctx, domain.CancelJobParams{JobID: req.GetId()})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CancelJobResponse{Job: jobToProto(job)}, nil
}
