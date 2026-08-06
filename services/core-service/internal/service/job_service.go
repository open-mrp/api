package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var jobSvcTracer = tracing.GetTracer("core-service.job_service")

// Caps the row errors a job carries. A bulk request takes up to 1000 rows and the job is
// polled in a loop, so an unbounded list would ride every response.
const maxRowErrors = 25

type jobSvcImpl struct {
	repos domain.RepoFactory
}

type JobSvcConfig struct {
	// Repos (required) is the repository factory. It is the only dependency: the
	// job service opens no transaction of its own, so it acts inside whichever one
	// this factory belongs to.
	Repos domain.RepoFactory
}

func (c *JobSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("job service: repos is required")
	}
	return nil
}

func NewJobSvc(config *JobSvcConfig) domain.JobSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &jobSvcImpl{repos: config.Repos}
}

type jobSvcFactoryImpl struct{}

// NewJobSvcFactory builds job services on demand for a caller's repository factory.
// A service that raises or runs jobs takes this rather than a JobSvc directly: the
// factory it holds is transactional inside a transaction and the root one outside,
// so the same call binds to whichever the caller is in.
func NewJobSvcFactory() domain.JobSvcFactory {
	return &jobSvcFactoryImpl{}
}

func (f *jobSvcFactoryImpl) Build(repos domain.RepoFactory) domain.JobSvc {
	return NewJobSvc(&JobSvcConfig{Repos: repos})
}

// GetJob retrieves a single job by ID, scoped to the account the caller is acting
// for, so a job ID from another tenant reads as absent rather than as someone else's
// work. Reading a job is the one thing a client does with one directly, so it is the
// one place a jobs permission is checked.
func (s *jobSvcImpl) GetJob(ctx context.Context, jobID string) (*domain.Job, *apierror.APIError) {
	ctx, span := jobSvcTracer.Start(ctx, "service.job.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkJobReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewJobRepo().Get(ctx, jobID, identity.Target.AccountID)
}

// GetJobForExecution reads a job for the worker running it. See domain.JobSvc for
// why this asks for no jobs permission where GetJob does. Both scope the read to the
// account the caller is acting for, so neither can reach another tenant's job.
func (s *jobSvcImpl) GetJobForExecution(ctx context.Context, jobID string) (*domain.Job, *apierror.APIError) {
	ctx, span := jobSvcTracer.Start(ctx, "service.job.get_for_execution")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	return s.repos.NewJobRepo().Get(ctx, jobID, identity.Target.AccountID)
}

// CreateJob raises a job to track a piece of asynchronous work, and returns it.
func (s *jobSvcImpl) CreateJob(ctx context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
	ctx, span := jobSvcTracer.Start(ctx, "service.job.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if !params.Type.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown job type.", "type"))
	}

	jobID, apiErr := id.GenID(id.JobIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	repo := s.repos.NewJobRepo()

	if apiErr := repo.Create(ctx, domain.CreateJobRepositoryParams{
		JobID:       jobID,
		JobItems:    params.JobItems,
		Type:        params.Type,
		AccountID:   accountID,
		CreatedByID: params.CreatedByID,
		Results:     params.Results,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	created, apiErr := repo.Get(ctx, jobID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := audit.NewPublisher().Publish(ctx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionCreate,
		ResourceType: constants.ObjectTypeJob,
		ResourceID:   created.ID,
		Changes:      audit.ComputeChanges(nil, created),
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return created, nil
}

// UpdateJob transitions a job to params.Status and returns it. Every lifecycle mark
// goes through here, so the invariants live here too.
//
// No endpoint reaches this: jobs are driven by the consumers carrying out the
// asynchronous work, and that work was authorized by the endpoint which enqueued it.
// So there is no permission check of its own to make.
//
// It takes no idempotency envelope either. A transition is already idempotent by
// construction — every column coalesces onto its stored value, so re-applying a mark
// keeps the first one rather than moving it — and the message inbox de-dupes
// redeliveries on top of that. An envelope would also be actively wrong here: it is
// keyed per request, and one message makes several transitions, so the second would
// replay the first's cached response instead of running.
//
// It opens no transaction of its own, writing through whichever repository factory
// this service was built with. A job service built from a transactional factory
// therefore settles the job inside the caller's transaction, which is what lets a
// completion commit with the work it reports.
func (s *jobSvcImpl) UpdateJob(ctx context.Context, params domain.UpdateJobServiceParams) (*domain.Job, *apierror.APIError) {
	ctx, span := jobSvcTracer.Start(ctx, "service.job.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	accountID := identity.Target.AccountID
	repo := s.repos.NewJobRepo()

	old, apiErr := repo.Get(ctx, params.JobID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if old.AccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("System jobs cannot be modified."))
	}
	if old.IsTerminal() {
		return nil, tracing.Trace(span, apierror.NewValidationError("This job has already finished and can no longer be modified."))
	}

	rowErrors, errorSummary := capRowErrors(params.Errors, params.ErrorSummary)
	repoParams := domain.UpdateJobRepositoryParams{
		JobID:        params.JobID,
		AccountID:    accountID,
		Results:      params.Results,
		Errors:       rowErrors,
		ErrorSummary: errorSummary,
	}
	if apiErr := stampTransition(&repoParams, params.Status, time.Now().UTC()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The update guards on the terminal timestamps, so it changes zero rows if the job
	// settled between the read above and here — a client cancel racing the worker's
	// completion. That is the authoritative terminal check: the read-based one above is
	// only a fast path. Whichever transition commits first wins; the loser lands here and
	// is refused, which for a completion rolls its write transaction back so no data lands.
	rows, apiErr := repo.Update(ctx, repoParams)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if rows == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("This job has already finished and can no longer be modified."))
	}

	updated, apiErr := repo.Get(ctx, params.JobID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := audit.NewPublisher().Publish(ctx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeJob,
		ResourceID:   updated.ID,
		Changes:      audit.ComputeChanges(old, updated),
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return updated, nil
}

// trims the failures to maxRowErrors, naming the full count in the summary so a truncated
// list cannot read as every row that failed. An existing summary is more useful, so it wins.
func capRowErrors(errs []apierror.RowError, summary *string) ([]apierror.RowError, *string) {
	if len(errs) <= maxRowErrors {
		return errs, summary
	}
	if summary == nil {
		s := fmt.Sprintf("%d rows failed; the first %d are listed.", len(errs), maxRowErrors)
		summary = &s
	}
	return errs[:maxRowErrors], summary
}

// stampTransition sets the one lifecycle timestamp that a transition to status
// derives from, leaving the others nil so the update preserves them. It is the
// single place a status becomes a timestamp, so the stored marks and the status
// Job.Status derives back out of them cannot disagree.
func stampTransition(params *domain.UpdateJobRepositoryParams, status constants.JobStatus, at time.Time) *apierror.APIError {
	switch status {
	case constants.JobStatusStarted:
		params.StartedAt = &at
	case constants.JobStatusCompleted:
		params.CompletedAt = &at
	case constants.JobStatusFailed:
		params.FailedAt = &at
	case constants.JobStatusCancelled:
		params.CancelledAt = &at
	case constants.JobStatusCreated:
		// The created state is the absence of every other timestamp; there is nothing to stamp.
	default:
		return apierror.NewValidationErrorWithParam("Unknown job status.", "status")
	}
	return nil
}

// The four transitions below name the marks a worker makes as it runs a job. Each is
// UpdateJob with the status filled in, so they inherit its checks rather than
// restating them.

// marks the job started and hands back the stamp. The caller needs it because that
// timestamp dates the work — UpdateJob already re-reads the row, so this costs nothing.
func (s *jobSvcImpl) StartJob(ctx context.Context, params domain.StartJobParams) (time.Time, *apierror.APIError) {
	updated, apiErr := s.UpdateJob(ctx, domain.UpdateJobServiceParams{
		JobID:  params.JobID,
		Status: constants.JobStatusStarted,
	})
	if apiErr != nil {
		return time.Time{}, apiErr
	}
	if updated.StartedAt == nil {
		return time.Time{}, apierror.NewInvariantViolationError("The job was started without a start time.")
	}
	return *updated.StartedAt, nil
}

func (s *jobSvcImpl) CompleteJob(ctx context.Context, params domain.CompleteJobParams) *apierror.APIError {
	_, apiErr := s.UpdateJob(ctx, domain.UpdateJobServiceParams{
		JobID:   params.JobID,
		Status:  constants.JobStatusCompleted,
		Results: params.Results,
		Errors:  params.Errors,
	})
	return apiErr
}

// failureReason renders the err into the reason stored on the job. The stored
// reason is part of a record the client reads back, so it takes the public message.
// Error() is the internal one — empty for a validation error, and full of internals
// for an internal error — so it is only ever the fallback. The internal detail stays
// on the caller's log line and span.
func failureReason(err *apierror.APIError) string {
	if err == nil {
		return ""
	}
	if err.PublicMessage != "" {
		return err.PublicMessage
	}
	return err.Error()
}

func (s *jobSvcImpl) FailJob(ctx context.Context, params domain.FailJobParams) {
	update := domain.UpdateJobServiceParams{
		JobID:  params.JobID,
		Status: constants.JobStatusFailed,
	}

	if reason := failureReason(params.ApiErr); reason != "" {
		update.ErrorSummary = &reason
		// The errors array is the per-item detail an executor that fails item by
		// item fills in, so a whole-job failure records the same entry shape —
		// exactly one, with no index because it names no row.
		update.Errors = []apierror.RowError{apierror.NewBatchError(params.ApiErr)}
	}

	if _, apiErr := s.UpdateJob(ctx, update); apiErr != nil {
		slog.WarnContext(ctx, "Failed to mark job failed", "error", apiErr, "job_id", params.JobID)
	}
}

// CancelJob is the one transition a client drives, so it is the one that authorizes.
// See domain.JobSvc for what cancelling does and does not stop.
func (s *jobSvcImpl) CancelJob(ctx context.Context, params domain.CancelJobParams) (*domain.Job, *apierror.APIError) {
	ctx, span := jobSvcTracer.Start(ctx, "service.job.cancel")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainJobs, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.UpdateJob(ctx, domain.UpdateJobServiceParams{
		JobID:  params.JobID,
		Status: constants.JobStatusCancelled,
	})
}

// checkJobReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need jobs:read for their own account, or customers:read / suppliers:read for external accounts.
func checkJobReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainJobs, types.ActionRead)
}
