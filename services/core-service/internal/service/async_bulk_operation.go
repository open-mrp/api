package service

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var asyncBulkTracer = tracing.GetTracer("core-service.async_bulk_operation")

// maxBulkRows caps a single bulk operation. Matches the per-endpoint validate tag.
const maxBulkRows = 1000

// asyncBulkDeps is the plumbing an async bulk operation runs on. A service supplies it
// from the dependencies it already holds; the engine never reaches past it.
type asyncBulkDeps struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

// jobs builds the job service bound to a given repository factory — transactional when
// that factory is a transaction's, the root one otherwise. See domain.JobSvcFactory.
func (d asyncBulkDeps) jobs(repos domain.RepoFactory) domain.JobSvc {
	return d.jobSvcFactory.Build(repos)
}

// BulkWriteResult is what an entity's Write returns: the job's results (the rows that
// succeeded) and, separately, its errors (the rows that failed), plus the flat list of
// written ids for AfterCommit. Both are domain values — encoding them is the
// repository's job, so nothing here serializes. Errors is nil when every row succeeded;
// Results is empty-but-non-nil when the write ran and nothing succeeded, which is how
// that is distinguished from a job that has recorded no results at all.
type BulkWriteResult struct {
	Results    []domain.RowResult
	Errors     []apierror.RowError
	WrittenIDs []string
}

// records a row that succeeded, as a create or an update
func newRowResult(index int, id string, isCreate bool) domain.RowResult {
	action := constants.JobResultActionUpdated
	if isCreate {
		action = constants.JobResultActionCreated
	}
	return domain.RowResult{Index: index, ID: id, Action: action}
}

// flattens the ids produced across a Write's results, for AfterCommit
func resultIDs(results []domain.RowResult) []string {
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

// bulkOperationSpec is everything an entity provides to run a bulk operation — create
// or upsert — as an async job. The engine owns the invariant plumbing (permissions
// scaffold, idempotency envelope, job lifecycle, the transaction, the outbox, recording
// results) and calls these hooks for the entity-specific parts. The variance (fuzzy
// resolution, id pre-generation, insert vs upsert, flow relinking) lives inside the
// hooks as plain functions, so it never has to fit a rigid interface.
//
// The 202 returns the canonical Job resource (`object: "job"`) for every async bulk
// operation, so there is no per-operation acknowledgment type — the raised job is the
// acknowledgment, and the client polls it via the Location header.
//
//   - TInput is the request row (fuzzy identifiers).
//   - TResolved is that row after Resolve (ids inline, plus pre-generated ids for a
//     create), which is what gets stored on the job and handed to Write.
type bulkOperationSpec[TInput, TResolved any] struct {
	// JobType tags the job so a polled job says what it did.
	JobType constants.JobType
	// RoutingKey is the command the accept phase enqueues and the consumer binds.
	RoutingKey contracts.AmqpRoutingKey
	// PermissionDomain is the domain checked in the accept phase.
	PermissionDomain types.PermissionDomain
	// Actions are the permissions required: {Create} for a create, {Create, Update}
	// for an upsert.
	Actions []types.Action
	// EntityName names the thing operated on, for the "no X provided" message.
	EntityName string

	// Validate runs in the accept phase before any database read: structural checks
	// and in-request duplicate rejection, with row-indexed params.
	Validate func(rows []TInput) *apierror.APIError
	// Resolve runs in the accept phase: it resolves every fuzzy reference to an id
	// (and, for a create, pre-generates the new rows' ids), failing fast with a
	// row-indexed 400, and returns the rows the job will store. Bad references
	// therefore fail synchronously rather than invisibly in the job.
	Resolve func(ctx context.Context, repos domain.RepoFactory, accountID string, rows []TInput) ([]TResolved, *apierror.APIError)
	// AcceptResults, when set, produces the results to record on the job at accept time —
	// a create returns its pre-generated ids here so the job the 202 returns already
	// carries them for the client to use immediately. An upsert leaves it nil: it cannot
	// know the created/updated split until Write runs, so its results fill in then.
	AcceptResults func(resolved []TResolved) []domain.RowResult
	// Write runs in the execute phase inside one transaction. It writes the resolved rows
	// row-by-row, wrapping each row's writes in sp.Run so a row that fails rolls back only
	// its own writes (partial success): the failure is recorded in the returned
	// BulkWriteResult.Errors and the remaining rows still commit. Write returns an error
	// only for an infrastructure failure that should roll the whole batch back and fail
	// the job (e.g. the initial bulk read). On redelivery a create converges on its
	// pre-generated ids and an upsert by natural key.
	Write func(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []TResolved) (BulkWriteResult, *apierror.APIError)
	// AfterCommit runs once the writes commit: side effects that must not roll them back, e.g. flow relinking. Return the failure — the engine traces it; the job is terminal by now, so nothing can be recorded on it.
	AfterCommit func(ctx context.Context, repos domain.RepoFactory, accountID string, writtenIDs []string) *apierror.APIError
}

// enqueueBulkOperation is the accept phase. It authorizes, validates and resolves
// synchronously, records the resolved payload on a job inside the outbox transaction,
// enqueues only the job's id, and returns the raised job — the canonical acknowledgment
// the 202 hands back and the client polls. No entity rows are written here.
func enqueueBulkOperation[TInput, TResolved any](
	ctx context.Context,
	deps asyncBulkDeps,
	spec bulkOperationSpec[TInput, TResolved],
	rows []TInput,
) (*domain.Job, *apierror.APIError) {
	ctx, span := asyncBulkTracer.Start(ctx, "service.async_bulk.enqueue."+spec.EntityName)
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, action := range spec.Actions {
		if apiErr := identity.CheckHasPermission(spec.PermissionDomain, action); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if len(rows) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("No "+spec.EntityName+" provided."))
	}
	if len(rows) > maxBulkRows {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot process more than 1000 "+spec.EntityName+" at a time."))
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	accountID := identity.Target.AccountID

	// Body-only, so it answers the same on every attempt: running it here means a malformed request never claims a key.
	if apiErr := spec.Validate(rows); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Resolve reads the database, so it runs after the key is claimed: otherwise a replay would re-resolve and could 400 on a reference deleted since the request was accepted.
	meds := deps.mediatorFactory.Build(deps.repos)
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Job](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Resolve fuzzy references now so a bad id fails with a row-indexed 400 rather than invisibly in the job. A failure leaves the key unfinished, so a retry can still succeed.
		resolved, apiErr := spec.Resolve(ctx, deps.repos, accountID, rows)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		jobItems, err := json.Marshal(resolved)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal bulk operation payload."))
		}

		// A create records its pre-generated ids on the job now, so the job the 202
		// returns already carries them; an upsert has no AcceptResults and fills its
		// results when the work runs.
		var acceptResults []domain.RowResult
		if spec.AcceptResults != nil {
			acceptResults = spec.AcceptResults(resolved)
		}

		// Attribute the job to the acting account user, best effort: not every actor
		// is one (an API key is not), and the attribution is advisory, so an actor that
		// does not resolve leaves it unset rather than failing the request.
		var createdByID *string
		if identity.Actor != nil && identity.Actor.ID != "" {
			if resolvedID, resolveErr := deps.repos.NewAccountUserRepo().ResolveAccountUserID(ctx, accountID, identity.Actor.ID); resolveErr == nil {
				createdByID = &resolvedID
			}
		}

		var raisedJob *domain.Job
		apiErr = deps.txManager.WithTx(ctx, func(txCtx context.Context, txRepos domain.RepoFactory) *apierror.APIError {
			// Record the resolved payload on a job and enqueue only that job's id. The
			// row is the single copy of the requested work, so the message stays a
			// constant size no matter how many rows were submitted, and the client has
			// something to poll for the outcome.
			job, apiErr := deps.jobs(txRepos).CreateJob(txCtx, domain.CreateJobServiceParams{
				JobItems:    jobItems,
				Type:        spec.JobType,
				CreatedByID: createdByID,
				Results:     acceptResults,
			})
			if apiErr != nil {
				return apiErr
			}
			raisedJob = job

			// Enqueue via the outbox inside the transaction so the command is published
			// if and only if the job and the acknowledgment commit.
			payloadJSON, err := json.Marshal(domain.BulkOperationJobEvent{JobID: job.ID})
			if err != nil {
				return apierror.NewInternalError(err, "Failed to marshal bulk operation job event.")
			}
			msg := contracts.AmqpMessage{Data: payloadJSON, Identity: identity}
			if requestID, ok := appctx.GetRequestID(txCtx); ok {
				msg.RequestID = requestID
			}
			if _, err := txRepos.NewOutboxRepo().Create(txCtx, messaging.OutboxMessageInput{
				ServiceName: domain.ServiceName,
				MessageType: string(spec.RoutingKey),
				Destination: messaging.ApplicationExchange,
				RoutingKey:  string(spec.RoutingKey),
				Payload:     msg,
			}); err != nil {
				return apierror.NewInternalError(err, "Failed to create bulk operation outbox message.")
			}

			return deps.mediatorFactory.Build(txRepos).Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, raisedJob)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return raisedJob, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// executeBulkOperation is the execute phase, called by the entity's consumer. It loads
// the job, runs the entity's Write inside one transaction, and settles the job in that
// same transaction so the rows and the record of having written them commit together.
// Exactly-once is provided by the message inbox, so there is no idempotency envelope
// here; the payload was validated and resolved at enqueue time.
func executeBulkOperation[TInput, TResolved any](
	ctx context.Context,
	deps asyncBulkDeps,
	spec bulkOperationSpec[TInput, TResolved],
	event domain.BulkOperationJobEvent,
) *apierror.APIError {
	ctx, span := asyncBulkTracer.Start(ctx, "service.async_bulk.execute."+spec.EntityName)
	defer span.End()

	if event.JobID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Bulk operation job event is missing a job."))
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	accountID := identity.Target.AccountID

	job, apiErr := deps.jobs(deps.repos).GetJobForExecution(ctx, event.JobID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// A job that already settled must not be executed again. The inbox de-dupes
	// redeliveries; this also covers a replay that outlives the inbox record.
	if job.IsTerminal() {
		return nil
	}

	var rows []TResolved
	if err := json.Unmarshal(job.JobItems, &rows); err != nil {
		apiErr := apierror.NewInternalError(err, "Job items are not a bulk operation payload.")
		deps.jobs(deps.repos).FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	// Marked outside the write transaction so the mark survives a rollback: a job that
	// failed mid-write should not read as never having run.
	if _, apiErr := deps.jobs(deps.repos).StartJob(ctx, domain.StartJobParams{JobID: job.ID}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	var writeResult BulkWriteResult
	apiErr = deps.txManager.WithTxSavepoint(ctx, func(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner) *apierror.APIError {
		res, apiErr := spec.Write(txCtx, txRepos, sp, accountID, rows)
		if apiErr != nil {
			return apiErr
		}
		writeResult = res

		// Settled through the transactional service, so the written rows and the record of
		// what happened — successes in results, per-row failures in errors — commit
		// together or not at all. A partly-failed batch still completes: Write only errors
		// on an infrastructure failure that should roll the whole batch back.
		return deps.jobs(txRepos).CompleteJob(txCtx, domain.CompleteJobParams{JobID: job.ID, Results: res.Results, Errors: res.Errors})
	})
	if apiErr != nil {
		deps.jobs(deps.repos).FailJob(ctx, domain.FailJobParams{JobID: job.ID, ApiErr: apiErr})
		return tracing.Trace(span, apiErr)
	}

	if spec.AfterCommit != nil {
		// Not returned: the job is already complete, so a failed side effect has nothing to roll back.
		if apiErr := spec.AfterCommit(ctx, deps.repos, accountID, writeResult.WrittenIDs); apiErr != nil {
			tracing.Trace(span, apiErr)
		}
	}

	return nil
}
