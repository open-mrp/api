package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// Every asynchronous operation raises and settles its job through JobSvc, so these
// rows pin what a caller cannot see for itself: which permission each entry point
// asks for, which lifecycle timestamp a transition writes, that a mark cannot land
// on a job that is not there, and what a failure ends up recording.
type JobSvcSystemSurfaceTestSuite struct {
	suite.Suite
	jobSvc      domain.JobSvc
	jobRepo     *repositorymock.MockJobRepo
	repoFactory *factorymock.MockRepoFactory
	ctrl        *gomock.Controller
}

func (suite *JobSvcSystemSurfaceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.jobRepo = repositorymock.NewMockJobRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewJobRepo().Return(suite.jobRepo).AnyTimes()
	// Raising or transitioning a job publishes an audit event through the outbox.
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.jobSvc = NewJobSvc(&JobSvcConfig{Repos: suite.repoFactory})
}

func (suite *JobSvcSystemSurfaceTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestJobSvcSystemSurfaceTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(JobSvcSystemSurfaceTestSuite))
}

// jobActorCtx builds an identity holding exactly the given permissions. The role is
// deliberately not admin: CheckHasPermission short-circuits to allow for admins, so
// an admin identity would pass every check here whether or not it exists, and prove
// nothing about which permission a method actually asks for.
func jobActorCtx(t *testing.T, accountID string, permissions map[string]bool) context.Context {
	t.Helper()
	roleType := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           genTestID(t, id.UserIDPrefix),
			AccountID:    &accountID,
			RoleType:     &roleType,
			Permissions:  permissions,
		},
	})
}

// storedJob is the job a transition reads before it writes.
func storedJob(jobID, accountID string) *domain.Job {
	return &domain.Job{ID: jobID, Type: constants.JobTypeBulkCreate, AccountID: &accountID}
}

// expectTransition stubs the read a transition makes, the reload it returns, and
// captures the patch written in between.
func (suite *JobSvcSystemSurfaceTestSuite) expectTransition(jobID, accountID string, existing *domain.Job, captured *domain.UpdateJobRepositoryParams) {
	if existing == nil {
		existing = storedJob(jobID, accountID)
	}
	gomock.InOrder(
		suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).Return(existing, nil).Times(1),
		suite.jobRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, params domain.UpdateJobRepositoryParams) (int64, *apierror.APIError) {
				*captured = params
				return 1, nil
			}).
			Times(1),
		// The re-read has to reflect the patch just written, or a caller reading back its
		// own transition — as StartJob does for the stamp — sees an unchanged job.
		suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).
			DoAndReturn(func(context.Context, string, string) (*domain.Job, *apierror.APIError) {
				updated := storedJob(jobID, accountID)
				updated.StartedAt = captured.StartedAt
				updated.CompletedAt = captured.CompletedAt
				updated.FailedAt = captured.FailedAt
				updated.CancelledAt = captured.CancelledAt
				return updated, nil
			}).Times(1),
	)
}

// --- CreateJob ---

// The job's ID is minted here, not by the caller: it is the job's identity. The
// caller acknowledges whatever comes back, so the ID handed out and the ID written
// have to be the same value. The account is taken from the identity rather than the
// params, so a job cannot be raised against an account the caller is not acting for.
func (suite *JobSvcSystemSurfaceTestSuite) TestCreateJob_MintsTheIDItReturns() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	createdBy := genTestID(suite.T(), id.AccountUserIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"production_runs:create": true})

	var written domain.CreateJobRepositoryParams
	suite.jobRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateJobRepositoryParams) *apierror.APIError {
			written = params
			return nil
		}).
		Times(1)
	suite.jobRepo.EXPECT().
		Get(gomock.Any(), gomock.Any(), accountID).
		DoAndReturn(func(_ context.Context, jobID, _ string) (*domain.Job, *apierror.APIError) {
			return &domain.Job{ID: jobID}, nil
		}).
		Times(1)

	job, apiErr := suite.jobSvc.CreateJob(ctx, domain.CreateJobServiceParams{
		Type:        constants.JobTypeBulkCreate,
		JobItems:    []byte(`{"runs":[]}`),
		CreatedByID: &createdBy,
	})

	suite.Nil(apiErr)
	suite.NotNil(job)
	suite.True(strings.HasPrefix(job.ID, string(id.JobIDPrefix)+"_"), "job id %q should carry the job prefix", job.ID)
	suite.Equal(job.ID, written.JobID)
	suite.Equal(constants.JobTypeBulkCreate, written.Type)
	suite.Equal(accountID, written.AccountID)
	suite.Equal(&createdBy, written.CreatedByID)
	suite.JSONEq(`{"runs":[]}`, string(written.JobItems))
}

// A job is raised by an operation that already authorized itself, so creating one
// asks for no jobs permission of its own: the identity here holds only the
// production_runs permission its bulk create was authorized with.
func (suite *JobSvcSystemSurfaceTestSuite) TestCreateJob_AsksForNoJobsPermission() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"production_runs:create": true})

	suite.jobRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	suite.jobRepo.EXPECT().
		Get(gomock.Any(), gomock.Any(), accountID).
		Return(&domain.Job{ID: genTestID(suite.T(), id.JobIDPrefix)}, nil).
		Times(1)

	job, apiErr := suite.jobSvc.CreateJob(ctx, domain.CreateJobServiceParams{
		Type:     constants.JobTypeBulkCreate,
		JobItems: []byte(`{"runs":[]}`),
	})

	suite.Nil(apiErr)
	suite.NotNil(job)
}

// The type reaches a varchar column, so an unknown one is rejected before it can be
// stored rather than read back later as a job nothing knows how to run.
func (suite *JobSvcSystemSurfaceTestSuite) TestCreateJob_RejectsUnknownType() {
	ctx := jobActorCtx(suite.T(), genTestID(suite.T(), id.AccountIDPrefix), nil)

	job, apiErr := suite.jobSvc.CreateJob(ctx, domain.CreateJobServiceParams{
		Type: constants.JobType("teleport"),
	})

	suite.Nil(job)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	suite.Equal("type", apiErr.Param)
}

// --- GetJob ---

// Reading a job is the one thing a client does with one directly, so it is the one
// place a jobs permission is asked for.
func (suite *JobSvcSystemSurfaceTestSuite) TestGetJob_RequiresReadPermission() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"production_runs:create": true})

	job, apiErr := suite.jobSvc.GetJob(ctx, jobID)

	suite.Nil(job)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// The worker reading the job it was handed is not exercising a permission — that was
// settled when the endpoint accepted the work. The identity here holds only the
// production_runs permission its bulk create was authorized with, exactly as the
// consumer restores it, and the read still has to succeed. If it did not, every actor
// allowed to start async work would need jobs:read too, and the ones without it would
// get a 202 and then silence.
func (suite *JobSvcSystemSurfaceTestSuite) TestGetJobForExecution_AsksForNoJobsPermission() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"production_runs:create": true})
	stored := storedJob(jobID, accountID)

	suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).Return(stored, nil).Times(1)

	job, apiErr := suite.jobSvc.GetJobForExecution(ctx, jobID)

	suite.Nil(apiErr)
	suite.Same(stored, job)
}

// Skipping the permission does not skip the tenant: the worker's read is scoped to
// the account it is acting for, the same as the client's.
func (suite *JobSvcSystemSurfaceTestSuite) TestGetJobForExecution_StillScopesToTheActingAccount() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	suite.jobRepo.EXPECT().
		Get(gomock.Any(), jobID, accountID).
		Return(nil, apierror.NewResourceNotFoundError("Job not found.")).
		Times(1)

	job, apiErr := suite.jobSvc.GetJobForExecution(ctx, jobID)

	suite.Nil(job)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

// The read is scoped to the account the caller is acting for, so a job ID from
// another tenant reads as absent rather than as someone else's work.
func (suite *JobSvcSystemSurfaceTestSuite) TestGetJob_ScopesToTheActingAccount() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"jobs:read": true})
	stored := storedJob(jobID, accountID)

	suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).Return(stored, nil).Times(1)

	job, apiErr := suite.jobSvc.GetJob(ctx, jobID)

	suite.Nil(apiErr)
	suite.Same(stored, job)
}

// --- Transitions ---

// There is no status column: Job.Status derives from these timestamps. So a
// transition must stamp exactly the one its status derives from and leave the rest
// nil, because a nil field is what tells the repository to preserve what is stored —
// stamping a second one would rewrite history.
func (suite *JobSvcSystemSurfaceTestSuite) TestTransitions_StampOnlyTheirOwnTimestamp() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	cases := []struct {
		name    string
		invoke  func(jobID string) *apierror.APIError
		stamped func(domain.UpdateJobRepositoryParams) *time.Time
		status  constants.JobStatus
	}{
		{
			name: "start",
			invoke: func(jobID string) *apierror.APIError {
				_, apiErr := suite.jobSvc.StartJob(ctx, domain.StartJobParams{JobID: jobID})
				return apiErr
			},
			stamped: func(p domain.UpdateJobRepositoryParams) *time.Time { return p.StartedAt },
			status:  constants.JobStatusStarted,
		},
		{
			name: "complete",
			invoke: func(jobID string) *apierror.APIError {
				return suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{JobID: jobID})
			},
			stamped: func(p domain.UpdateJobRepositoryParams) *time.Time { return p.CompletedAt },
			status:  constants.JobStatusCompleted,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			jobID := genTestID(suite.T(), id.JobIDPrefix)
			var captured domain.UpdateJobRepositoryParams
			suite.expectTransition(jobID, accountID, nil, &captured)

			apiErr := tc.invoke(jobID)

			suite.Nil(apiErr)
			suite.Equal(jobID, captured.JobID)
			suite.Equal(accountID, captured.AccountID)
			suite.NotNil(tc.stamped(captured), "%s should stamp its own timestamp", tc.name)

			// Reading the patch back as a job proves the mark is the one the
			// derived status will actually report.
			job := &domain.Job{
				StartedAt:   captured.StartedAt,
				CompletedAt: captured.CompletedAt,
				FailedAt:    captured.FailedAt,
				CancelledAt: captured.CancelledAt,
			}
			suite.Equal(tc.status, job.Status())
		})
	}
}

// The update is a blind UPDATE ... WHERE job_id AND account_id: against a job that is
// absent, or another tenant's, it matches no rows and reports no error. Without the
// read first, a mark aimed at the wrong job would look like it landed.
func (suite *JobSvcSystemSurfaceTestSuite) TestTransitions_RejectAMarkOnAJobThatIsNotThere() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	// No Update is stubbed: reaching one would fail this test.
	suite.jobRepo.EXPECT().
		Get(gomock.Any(), jobID, accountID).
		Return(nil, apierror.NewResourceNotFoundError("Job not found.")).
		Times(1)

	_, apiErr := suite.jobSvc.StartJob(ctx, domain.StartJobParams{JobID: jobID})

	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

// A settled job is done being written to.
func (suite *JobSvcSystemSurfaceTestSuite) TestTransitions_RejectASettledJob() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	settled := storedJob(jobID, accountID)
	completedAt := time.Now().UTC()
	settled.CompletedAt = &completedAt

	suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).Return(settled, nil).Times(1)

	apiErr := suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{JobID: jobID})

	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// CompleteJob carries the outcome a client polling the job reads back.
func (suite *JobSvcSystemSurfaceTestSuite) TestCompleteJob_RecordsResults() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	var captured domain.UpdateJobRepositoryParams
	suite.expectTransition(jobID, accountID, nil, &captured)

	results := []domain.RowResult{{Index: 0, ID: "pr_1", Status: constants.JobResultStatusCreated, ResourceType: constants.ObjectTypeProductionRun}}

	apiErr := suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{
		JobID:   jobID,
		Results: results,
	})

	suite.Nil(apiErr)
	suite.NotNil(captured.CompletedAt)
	suite.Equal(results, captured.Results, "the completion carries its results through to the repository")
}

// --- CancelJob ---

// Cancelling is the one transition a client drives, so it is the one that asks for a
// permission. It also returns the job rather than nothing: the row is not removed,
// and its new state is what the caller asked for.
func (suite *JobSvcSystemSurfaceTestSuite) TestCancelJob_SettlesTheJobAndReturnsIt() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"jobs:delete": true})

	var captured domain.UpdateJobRepositoryParams
	suite.expectTransition(jobID, accountID, nil, &captured)

	job, apiErr := suite.jobSvc.CancelJob(ctx, domain.CancelJobParams{JobID: jobID})

	suite.Nil(apiErr)
	suite.NotNil(job)
	suite.Equal(jobID, job.ID)
	suite.NotNil(captured.CancelledAt)

	cancelled := &domain.Job{CancelledAt: captured.CancelledAt}
	suite.Equal(constants.JobStatusCancelled, cancelled.Status())
	suite.True(cancelled.IsTerminal())
}

// Reading a job is not license to stop it. The identity here can see the job but
// holds no jobs:delete.
func (suite *JobSvcSystemSurfaceTestSuite) TestCancelJob_RequiresDeletePermission() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, map[string]bool{"jobs:read": true})

	// No repository call is stubbed: reaching one would fail this test.
	job, apiErr := suite.jobSvc.CancelJob(ctx, domain.CancelJobParams{JobID: jobID})

	suite.Nil(job)
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// Cancelling settles the job, and a settled job refuses further marks. That is what
// stops in-flight work from committing: the completion is the last thing inside the
// writes' transaction, so refusing it rolls that transaction back.
func (suite *JobSvcSystemSurfaceTestSuite) TestCancelJob_StopsAnInFlightJobFromCompleting() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	cancelledAt := time.Now().UTC()
	startedAt := cancelledAt.Add(-time.Minute)
	inFlightThenCancelled := storedJob(jobID, accountID)
	inFlightThenCancelled.StartedAt = &startedAt
	inFlightThenCancelled.CancelledAt = &cancelledAt

	suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).Return(inFlightThenCancelled, nil).Times(1)

	apiErr := suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{JobID: jobID})

	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// The race the pre-read cannot catch: the job still looks live when the transition
// reads it, but a concurrent cancel settles it before the guarded UPDATE runs, so the
// UPDATE matches no row. Zero rows affected is the authoritative "already settled" — the
// completion is refused, which rolls its write transaction back so no data is committed.
func (suite *JobSvcSystemSurfaceTestSuite) TestTransitions_RefuseWhenGuardedUpdateMatchesNoRow() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	// The pre-read sees a started (non-terminal) job, so it passes the fast-path check...
	started := storedJob(jobID, accountID)
	startedAt := time.Now().UTC()
	started.StartedAt = &startedAt
	suite.jobRepo.EXPECT().Get(gomock.Any(), jobID, accountID).Return(started, nil).Times(1)
	// ...but the guarded UPDATE finds the row already settled and changes nothing. No
	// reload Get follows: the transition returns on the zero-row result.
	suite.jobRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(int64(0), nil).Times(1)

	apiErr := suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{JobID: jobID})

	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// --- FailJob ---

// What a failure records is the whole reason FailJob takes the cause instead of a
// pre-rendered string. A validation error carries its message publicly and leaves
// Error() empty, so summarizing from Error() stores a blank reason — which is what
// this repository did before the cause was handed over whole.
func (suite *JobSvcSystemSurfaceTestSuite) TestFailJob_RecordsAReadableReason() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	cases := []struct {
		name  string
		cause *apierror.APIError
		want  string
	}{
		{
			name:  "validation error records its public message",
			cause: apierror.NewValidationError(`Quantity value "many" is not a decimal.`),
			want:  `Quantity value "many" is not a decimal.`,
		},
		{
			// The internal detail names internals, so the job — which the client
			// reads — gets the public message. The detail stays on the span.
			name:  "internal error records the public message, not the internals",
			cause: apierror.NewInternalError(errors.New("dial tcp 10.0.0.1: connection refused"), "job items unreadable"),
			want:  "Something went wrong.",
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			jobID := genTestID(suite.T(), id.JobIDPrefix)
			var captured domain.UpdateJobRepositoryParams
			suite.expectTransition(jobID, accountID, nil, &captured)

			suite.jobSvc.FailJob(ctx, domain.FailJobParams{JobID: jobID, ApiErr: tc.cause})

			suite.NotNil(captured.FailedAt)
			// A whole-job failure names no row, so it settles on the job as the cause's
			// canonical ResponseError. The internals must not survive the rendering.
			suite.Require().NotNil(captured.Error)
			suite.Equal(tc.want, captured.Error.Message)
			suite.NotContains(string(marshalJSON(captured.Error)), "connection refused")
			// It is the job that failed, not any one row, so no row is invented for it.
			suite.Empty(captured.Results)
		})
	}
}

// A failed job stays retryable: the inbox redelivers on error, and that retry has to
// be allowed to run. Only a settled job is terminal.
func (suite *JobSvcSystemSurfaceTestSuite) TestFailJob_LeavesTheJobRetryable() {
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)

	var captured domain.UpdateJobRepositoryParams
	suite.expectTransition(jobID, accountID, nil, &captured)

	suite.jobSvc.FailJob(ctx, domain.FailJobParams{
		JobID:  jobID,
		ApiErr: apierror.NewValidationError("write failed"),
	})

	failed := &domain.Job{StartedAt: captured.StartedAt, FailedAt: captured.FailedAt}
	suite.Equal(constants.JobStatusFailed, failed.Status())
	suite.False(failed.IsTerminal())
}

// A thousand-row batch must not put a thousand entries on a job the client polls in a
// loop. The record says it was trimmed, so a short list cannot read as the whole story.
func (suite *JobSvcSystemSurfaceTestSuite) TestCompleteJob_CapsTheRowResultsItRecords() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)
	jobID := genTestID(suite.T(), id.JobIDPrefix)

	// Failures sit at the end of the request, behind more written rows than the cap
	// allows, so a naive head-of-list trim would drop every one of them.
	results := make([]domain.RowResult, 0, 1000)
	for i := range 1000 {
		row := domain.RowResult{Index: i, ID: "pr_1", Status: constants.JobResultStatusCreated}
		if i >= 990 {
			responseErr := apierror.NewValidationError("bad row").ToResponseError()
			row = domain.RowResult{Index: i, Status: constants.JobResultStatusFailed, Error: &responseErr}
		}
		results = append(results, row)
	}

	var captured domain.UpdateJobRepositoryParams
	suite.expectTransition(jobID, accountID, nil, &captured)

	suite.Require().Nil(suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{JobID: jobID, Results: results}))

	suite.Len(captured.Results, maxRowResults)
	suite.True(captured.ResultsTruncated, "a trimmed record must say so")

	// Every failure survives the cut — the reason a row was rejected is worth more than
	// an id the client can re-read — and what is kept stays in request order.
	var failedCount int
	for i, r := range captured.Results {
		if r.Failed() {
			failedCount++
		}
		if i > 0 && captured.Results[i-1].Index >= r.Index {
			suite.Failf("the kept rows must stay in index order", "%d then %d", captured.Results[i-1].Index, r.Index)
		}
	}
	suite.Equal(10, failedCount)
}

// A batch inside the cap is recorded whole, and says nothing it does not need to.
func (suite *JobSvcSystemSurfaceTestSuite) TestCompleteJob_KeepsEveryRowUnderTheCap() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	ctx := jobActorCtx(suite.T(), accountID, nil)
	jobID := genTestID(suite.T(), id.JobIDPrefix)

	results := []domain.RowResult{{Index: 0, ID: "pr_1", Status: constants.JobResultStatusCreated}}

	var captured domain.UpdateJobRepositoryParams
	suite.expectTransition(jobID, accountID, nil, &captured)

	suite.Require().Nil(suite.jobSvc.CompleteJob(ctx, domain.CompleteJobParams{JobID: jobID, Results: results}))

	suite.Equal(results, captured.Results)
	suite.False(captured.ResultsTruncated)
}
