package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	servicemock "github.com/open-mrp/api/services/core-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// splitJobResults reads a Write's results into created and updated id slices in request
// order — the form the entity tests assert on. It is the row-indexed results' inverse of
// the old created_ids/updated_ids split. No decoding: results are domain values now, and
// the JSON of the column they land in is the repository's business.
func splitJobResults(results []domain.RowResult) (created, updated []string) {
	for _, r := range results {
		switch r.Status {
		case constants.JobResultStatusCreated:
			created = append(created, r.ID)
		case constants.JobResultStatusUpdated:
			updated = append(updated, r.ID)
		}
	}
	return created, updated
}

// The async bulk-operation engine is what every converted bulk create/upsert runs
// through, so these rows pin its orchestration directly, with a trivial spec standing
// in for a real entity: the accept phase records the resolved rows on a job, enqueues
// its id, and returns the acknowledgment; the execute phase loads that job, writes
// through the spec, and settles the job — starting before the write and completing
// inside its transaction, failing outside it.
type AsyncBulkEngineTestSuite struct {
	suite.Suite
	deps            asyncBulkDeps
	repoFactory     *factorymock.MockRepoFactory
	accountUserRepo *repositorymock.MockAccountUserRepo
	idempotencyMed  *mediatormock.MockIdempotencyMed
	jobSvc          *servicemock.MockJobSvc
	outbox          *stubOutboxRepo
	ctrl            *gomock.Controller
}

func (suite *AsyncBulkEngineTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.outbox = &stubOutboxRepo{}
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(suite.outbox).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: suite.idempotencyMed}).AnyTimes()

	suite.jobSvc = servicemock.NewMockJobSvc(suite.ctrl)
	jobSvcFactory := factorymock.NewMockJobSvcFactory(suite.ctrl)
	jobSvcFactory.EXPECT().Build(gomock.Any()).Return(suite.jobSvc).AnyTimes()

	suite.deps = asyncBulkDeps{
		repos:           suite.repoFactory,
		mediatorFactory: mediatorFactory,
		jobSvcFactory:   jobSvcFactory,
		txManager:       &stubTxManager{factory: suite.repoFactory},
	}
}

func (suite *AsyncBulkEngineTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestAsyncBulkEngineTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AsyncBulkEngineTestSuite))
}

// fakeRow stands in for an entity's input/resolved row. Resolve is a passthrough, so
// the same type serves both.
type fakeRow struct {
	Name string `json:"name"`
}

// fakeSpec builds a spec whose hooks record what the engine handed them. Write returns
// the given created/updated ids (as an upsert result) and per-row errors JSON, or the
// given fatal error.
func fakeSpec(
	validateErr *apierror.APIError,
	writeCreated, writeUpdated []string,
	writeErr *apierror.APIError,
	writeErrors []apierror.RowError,
	writeSeen *[]fakeRow,
	afterCommitSeen *[]string,
) bulkOperationSpec[fakeRow, fakeRow] {
	return bulkOperationSpec[fakeRow, fakeRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeProductionStep,
		RoutingKey:       "core.cmd.fake",
		PermissionDomain: types.PermissionDomainProductionSteps,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "fakes",
		Validate: func(rows []fakeRow) *apierror.APIError {
			return validateErr
		},
		Resolve: func(_ context.Context, _ domain.RepoFactory, _ string, rows []fakeRow) ([]fakeRow, *apierror.APIError) {
			return rows, nil
		},
		Write: func(_ context.Context, _ domain.RepoFactory, _ db.SavepointRunner, _ string, rows []fakeRow) (BulkWriteResult, *apierror.APIError) {
			if writeSeen != nil {
				*writeSeen = rows
			}
			if writeErr != nil {
				return BulkWriteResult{}, writeErr
			}
			results := make([]domain.RowResult, 0, len(writeCreated)+len(writeUpdated))
			for _, cid := range writeCreated {
				results = append(results, newRowResult(len(results), cid, true))
			}
			for _, uid := range writeUpdated {
				results = append(results, newRowResult(len(results), uid, false))
			}
			return BulkWriteResult{
				Results:    results,
				Errors:     writeErrors,
				WrittenIDs: append(append([]string{}, writeCreated...), writeUpdated...),
			}, nil
		},
		AfterCommit: func(_ context.Context, _ domain.RepoFactory, _ string, writtenIDs []string) *apierror.APIError {
			if afterCommitSeen != nil {
				*afterCommitSeen = writtenIDs
			}
			return nil
		},
	}
}

func (suite *AsyncBulkEngineTestSuite) startedKey() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_fake", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).
		Times(1)
}

// --- Accept phase ---

// The accept phase records the resolved rows on a job and hands back an acknowledgment
// carrying that job's id. The rows must reach the job intact — it is the single copy of
// the requested work.
func (suite *AsyncBulkEngineTestSuite) TestEnqueue_RecordsResolvedRowsOnAJob() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)

	suite.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(genTestID(suite.T(), id.AccountUserIDPrefix), nil).AnyTimes()
	suite.startedKey()

	var recorded domain.CreateJobServiceParams
	suite.jobSvc.EXPECT().
		CreateJob(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
			recorded = params
			return &domain.Job{ID: jobID}, nil
		}).
		Times(1)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_fake", gomock.Any()).Return(nil).Times(1)

	rows := []fakeRow{{Name: "a"}, {Name: "b"}}
	ack, apiErr := enqueueBulkOperation(ctx, suite.deps, fakeSpec(nil, nil, nil, nil, nil, nil, nil), rows)

	suite.Nil(apiErr)
	suite.NotNil(ack)
	suite.Equal(jobID, ack.ID)
	suite.Equal(constants.JobTypeBulkUpsert, recorded.Type)

	var stored []fakeRow
	suite.NoError(json.Unmarshal(recorded.JobItems, &stored))
	suite.Equal(rows, stored)
}

// An API key is not an account user, so attribution falls through to the key's own id
// rather than leaving created_by empty — retrieve-job expands it as an actor.
func (suite *AsyncBulkEngineTestSuite) TestEnqueue_AttributesAnAPIKeyAsCreatedBy() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	actorID := genTestID(suite.T(), id.APIKeyIDPrefix)
	roleCode := string(constants.RoleTypeAdmin)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           actorID,
			AccountID:    &accountID,
			RoleType:     &roleCode,
			Permissions:  map[string]bool{"production_steps:create": true, "production_steps:update": true},
		},
	})

	suite.accountUserRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), accountID, actorID).
		Return("", apierror.NewResourceNotFoundError("not an account user")).
		Times(1)
	suite.startedKey()

	var recorded domain.CreateJobServiceParams
	suite.jobSvc.EXPECT().
		CreateJob(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
			recorded = params
			return &domain.Job{ID: jobID}, nil
		}).
		Times(1)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_fake", gomock.Any()).Return(nil).Times(1)

	ack, apiErr := enqueueBulkOperation(ctx, suite.deps, fakeSpec(nil, nil, nil, nil, nil, nil, nil), []fakeRow{{Name: "a"}})

	suite.Nil(apiErr)
	suite.NotNil(ack)
	suite.Require().NotNil(recorded.CreatedByID)
	suite.Equal(actorID, *recorded.CreatedByID)
}

// A validation error stops the accept phase before any job is raised.
func (suite *AsyncBulkEngineTestSuite) TestEnqueue_ValidationFailureRaisesNoJob() {
	ctx := internalProductionStepCtx(genTestID(suite.T(), id.AccountIDPrefix))

	// No CreateJob / idempotency is stubbed: reaching either would fail this test.
	badRows := []fakeRow{{Name: "bad"}}
	ack, apiErr := enqueueBulkOperation(ctx, suite.deps,
		fakeSpec(apierror.NewValidationError("bad row"), nil, nil, nil, nil, nil, nil), badRows)

	suite.Nil(ack)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// A replay hands back the job the first attempt raised, even when the references it used are gone: the key is read before anything resolves.
func (suite *AsyncBulkEngineTestSuite) TestEnqueue_ReplayDoesNotResolveAgain() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)

	code := 202
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_fake",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(&domain.Job{ID: jobID}),
		}, nil).
		Times(1)
	// No CreateJob is stubbed: a replay must raise nothing.

	// A resolution that would now fail, and must never be reached.
	resolveCalled := false
	spec := fakeSpec(nil, nil, nil, nil, nil, nil, nil)
	spec.Resolve = func(context.Context, domain.RepoFactory, string, []fakeRow) ([]fakeRow, *apierror.APIError) {
		resolveCalled = true
		return nil, apierror.NewValidationError("the referenced row has since been deleted")
	}

	ack, apiErr := enqueueBulkOperation(ctx, suite.deps, spec, []fakeRow{{Name: "a"}})

	suite.Nil(apiErr)
	suite.NotNil(ack)
	suite.Equal(jobID, ack.ID, "the replay returns the job the first attempt raised")
	suite.False(resolveCalled, "a replay must not resolve the request again")
}

// An empty batch is rejected up front.
func (suite *AsyncBulkEngineTestSuite) TestEnqueue_EmptyRejected() {
	ctx := internalProductionStepCtx(genTestID(suite.T(), id.AccountIDPrefix))

	ack, apiErr := enqueueBulkOperation(ctx, suite.deps, fakeSpec(nil, nil, nil, nil, nil, nil, nil), []fakeRow{})

	suite.Nil(ack)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// --- Execute phase ---

func (suite *AsyncBulkEngineTestSuite) loadJob(jobID, accountID string, rows []fakeRow, existing *domain.Job) {
	if existing == nil {
		existing = &domain.Job{ID: jobID, Type: constants.JobTypeBulkUpsert, AccountID: &accountID}
	}
	existing.JobItems = marshalJSON(rows)
	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), jobID).Return(existing, nil).Times(1)
}

// Execute loads the job, writes the rows it stored, and settles it — starting before
// the write and completing with the results the write produced.
func (suite *AsyncBulkEngineTestSuite) TestExecute_WritesRowsThenCompletesWithResults() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)
	rows := []fakeRow{{Name: "a"}}

	suite.loadJob(jobID, accountID, rows, nil)

	var order []string
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), domain.StartJobParams{JobID: jobID}).
		DoAndReturn(func(context.Context, domain.StartJobParams) (time.Time, *apierror.APIError) {
			order = append(order, "start")
			return time.Now(), nil
		}).Times(1)

	var completeResults []domain.RowResult
	suite.jobSvc.EXPECT().CompleteJob(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CompleteJobParams) *apierror.APIError {
			order = append(order, "complete")
			completeResults = params.Results
			return nil
		}).Times(1)

	var writeSeen []fakeRow
	var afterSeen []string
	spec := fakeSpec(nil, []string{"cr_1"}, []string{"up_1"}, nil, nil, &writeSeen, &afterSeen)

	apiErr := executeBulkOperation(ctx, suite.deps, spec, domain.BulkOperationJobEvent{JobID: jobID})

	suite.Nil(apiErr)
	suite.Equal([]string{"start", "complete"}, order)
	suite.Equal(rows, writeSeen)

	created, updated := splitJobResults(completeResults)
	suite.Equal([]string{"cr_1"}, created)
	suite.Equal([]string{"up_1"}, updated)
	suite.Empty(failedRows(completeResults), "no row failed, so every entry names a resource")
	// AfterCommit sees every written id.
	suite.ElementsMatch([]string{"cr_1", "up_1"}, afterSeen)
}

// AfterCommit runs once the writes are committed and the job is completed, so a failure
// there has nothing left to roll back — it must not fail the job or the delivery. The
// engine reports it instead (log and span), which is why the hook returns its error
// rather than deciding what to do with it: a hook that handled its own failure is how
// one gets swallowed.
func (suite *AsyncBulkEngineTestSuite) TestExecute_AfterCommitFailureLeavesTheJobCompleted() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)

	suite.loadJob(jobID, accountID, []fakeRow{{Name: "a"}}, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(time.Now(), nil).Times(1)
	suite.jobSvc.EXPECT().CompleteJob(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	// No FailJob is stubbed: reaching it would fail this test.

	spec := fakeSpec(nil, []string{"cr_1"}, nil, nil, nil, nil, nil)
	spec.AfterCommit = func(context.Context, domain.RepoFactory, string, []string) *apierror.APIError {
		return apierror.NewInternalError(nil, "flow relinking failed")
	}

	apiErr := executeBulkOperation(ctx, suite.deps, spec, domain.BulkOperationJobEvent{JobID: jobID})

	suite.Nil(apiErr, "a post-commit side effect must not fail the job or redeliver the message")
}

// rowErrIndex reads an entry's row index. Every collected failure names one now — a
// failure that names no row settles on the job instead of among its rows.
func rowErrIndex(e apierror.RowError) int {
	return e.Index
}

// rowErrMessage reads an entry's public message.
func rowErrMessage(e apierror.RowError) string {
	return e.Error.Message
}

// failedRows picks the rejected entries out of a job's row outcomes.
func failedRows(results []domain.RowResult) []domain.RowResult {
	var out []domain.RowResult
	for _, r := range results {
		if r.Failed() {
			out = append(out, r)
		}
	}
	return out
}

// Partial success: a batch with some failed rows still completes. The engine merges the
// write's successes and failures into the job's one row-indexed list, in request order,
// and the job is completed (not failed).
func (suite *AsyncBulkEngineTestSuite) TestExecute_PartialFailureCompletesWithErrors() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)

	suite.loadJob(jobID, accountID, []fakeRow{{Name: "a"}, {Name: "b"}}, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(time.Now(), nil).Times(1)

	var completeResults []domain.RowResult
	suite.jobSvc.EXPECT().CompleteJob(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CompleteJobParams) *apierror.APIError {
			completeResults = params.Results
			return nil
		}).Times(1)
	// No FailJob is stubbed: a partly-failed batch must complete, not fail.

	rowErrs := []apierror.RowError{apierror.NewRowError(1, apierror.NewValidationError("row b failed"))}
	spec := fakeSpec(nil, []string{"cr_1"}, nil, nil, rowErrs, nil, nil)

	apiErr := executeBulkOperation(ctx, suite.deps, spec, domain.BulkOperationJobEvent{JobID: jobID})

	suite.Nil(apiErr)
	// Every submitted row is accounted for in the one list, ordered by index.
	suite.Len(completeResults, 2)
	suite.Equal(0, completeResults[0].Index)
	suite.Equal(constants.JobResultStatusCreated, completeResults[0].Status)
	// The engine stamps what the operation writes onto the rows that wrote.
	suite.Equal(constants.ObjectTypeProductionStep, completeResults[0].ResourceType)

	failed := failedRows(completeResults)
	suite.Len(failed, 1)
	suite.Equal(1, failed[0].Index)
	suite.Require().NotNil(failed[0].Error)
	suite.Equal("row b failed", failed[0].Error.Message)
	// A rejected row wrote nothing, so it names no resource.
	suite.Empty(failed[0].ID)
	suite.Empty(failed[0].ResourceType)
}

// A write failure fails the job and surfaces the cause; the failure mark is written
// outside the rolled-back transaction.
func (suite *AsyncBulkEngineTestSuite) TestExecute_WriteFailureFailsTheJob() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)
	writeErr := apierror.NewValidationError("write failed")

	suite.loadJob(jobID, accountID, []fakeRow{{Name: "a"}}, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(time.Now(), nil).Times(1)
	// No CompleteJob is stubbed: reaching it would fail this test.
	var failed []domain.FailJobParams
	suite.jobSvc.EXPECT().FailJob(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, params domain.FailJobParams) { failed = append(failed, params) }).Times(1)

	apiErr := executeBulkOperation(ctx, suite.deps, fakeSpec(nil, nil, nil, writeErr, nil, nil, nil), domain.BulkOperationJobEvent{JobID: jobID})

	suite.NotNil(apiErr)
	suite.Same(writeErr, apiErr)
	suite.Len(failed, 1)
	suite.Same(writeErr, failed[0].ApiErr)
}

// A job that already settled is not executed again: no write is stubbed, so any attempt
// to run the spec would fail this test.
func (suite *AsyncBulkEngineTestSuite) TestExecute_SettledJobIsSkipped() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	ctx := internalProductionStepCtx(accountID)

	completedAt := time.Now().UTC()
	suite.loadJob(jobID, accountID, []fakeRow{{Name: "a"}}, &domain.Job{
		ID:          jobID,
		Type:        constants.JobTypeBulkUpsert,
		AccountID:   &accountID,
		CompletedAt: &completedAt,
	})

	var writeCalled bool
	spec := fakeSpec(nil, nil, nil, nil, nil, nil, nil)
	spec.Write = func(context.Context, domain.RepoFactory, db.SavepointRunner, string, []fakeRow) (BulkWriteResult, *apierror.APIError) {
		writeCalled = true
		return BulkWriteResult{}, nil
	}

	apiErr := executeBulkOperation(ctx, suite.deps, spec, domain.BulkOperationJobEvent{JobID: jobID})

	suite.Nil(apiErr)
	suite.False(writeCalled)
}
