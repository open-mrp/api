package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	servicemock "github.com/augno/api/services/core-service/internal/domain/mock/service"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// The bulk create endpoint is exercised end-to-end in
// tests/e2e/api/bulk_create_production_runs_test.go — happy paths, unknown
// references, caps, and idempotent replay all live there. This suite only covers
// what e2e cannot reach: forged identities, repository and outbox failures,
// idempotency recovery internals, and the defense-in-depth decimal checks that the
// gateway's float64 binding makes unreachable from the outside.
type ProductionRunBulkCreateTestSuite struct {
	suite.Suite
	productionRunSvc        domain.ProductionRunSvc
	accountUserRepo         *repositorymock.MockAccountUserRepo
	productionRunRepo       *repositorymock.MockProductionRunRepo
	batchRepo               *repositorymock.MockBatchRepo
	itemRepo                *repositorymock.MockItemRepo
	unitRepo                *repositorymock.MockUnitRepo
	productionStepQueryRepo *repositorymock.MockProductionStepQueryRepo
	scanningStationRepo     *repositorymock.MockScanningStationRepo
	repoFactory             *factorymock.MockRepoFactory
	mediatorFactory         *factorymock.MockMediatorFactory
	idempotencyMed          *mediatormock.MockIdempotencyMed
	jobSvc                  *servicemock.MockJobSvc
	outboxRepo              messaging.OutboxRepo
	ctrl                    *gomock.Controller
	// actorID is the user the test identity acts as. It is fixed per test so the
	// job's created_by lookup can be expected by ID rather than by a catch-all,
	// which would shadow the responsible-user lookups of sibling subtests.
	actorID string
}

// failingOutboxRepo simulates an outbox write failure inside a transaction.
type failingOutboxRepo struct{}

func (r *failingOutboxRepo) Create(_ context.Context, _ messaging.OutboxMessageInput) (int64, error) {
	return 0, errors.New("outbox insert failed")
}

func (suite *ProductionRunBulkCreateTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.actorID = genTestID(suite.T(), id.UserIDPrefix)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.productionRunRepo = repositorymock.NewMockProductionRunRepo(suite.ctrl)
	suite.batchRepo = repositorymock.NewMockBatchRepo(suite.ctrl)
	suite.itemRepo = repositorymock.NewMockItemRepo(suite.ctrl)
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	suite.productionStepQueryRepo = repositorymock.NewMockProductionStepQueryRepo(suite.ctrl)
	suite.scanningStationRepo = repositorymock.NewMockScanningStationRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewProductionRunRepo().Return(suite.productionRunRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewBatchRepo().Return(suite.batchRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewItemRepo().Return(suite.itemRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewProductionStepQueryRepo().Return(suite.productionStepQueryRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewScanningStationRepo().Return(suite.scanningStationRepo).AnyTimes()
	suite.outboxRepo = &stubOutboxRepo{}
	suite.repoFactory.EXPECT().NewOutboxRepo().DoAndReturn(func() messaging.OutboxRepo {
		return suite.outboxRepo
	}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	// The service reaches jobs only through this factory, so stubbing it here is
	// what keeps the job's storage out of these tests: they assert what the bulk
	// create asks of the job service, not how a job row is written.
	suite.jobSvc = servicemock.NewMockJobSvc(suite.ctrl)
	jobSvcFactory := factorymock.NewMockJobSvcFactory(suite.ctrl)
	jobSvcFactory.EXPECT().Build(gomock.Any()).Return(suite.jobSvc).AnyTimes()

	suite.productionRunSvc = NewProductionRunSvc(&ProductionRunSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *ProductionRunBulkCreateTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestProductionRunBulkCreateTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProductionRunBulkCreateTestSuite))
}

// genTestID generates a well-formed ID with the given prefix via the real ID
// generator — test IDs are never hand-written so they cannot drift from the actual
// ID format.
func genTestID(t *testing.T, prefix id.IDPrefix) string {
	t.Helper()
	generated, apiErr := id.GenID(prefix, nil)
	if apiErr != nil {
		t.Fatalf("failed to generate id with prefix %q: %v", prefix, apiErr)
	}
	return generated
}

func internalProductionRunCtx(t *testing.T, accountID, actorID string) context.Context {
	t.Helper()
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           actorID,
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"production_runs:read":   true,
				"production_runs:create": true,
				"production_runs:update": true,
				"production_runs:delete": true,
			},
		},
	})
}

func validBulkCreateProductionRunParams(t *testing.T) domain.BulkCreateProductionRunParams {
	t.Helper()
	return domain.BulkCreateProductionRunParams{
		ResponsibleUserID: genTestID(t, id.UserIDPrefix),
		Batches: []domain.BulkCreateBatchParams{
			{Item: domain.ItemIdentifier{SKU: "SKU-TEST-1"}, QuantityValue: "100", QuantityUnit: domain.UnitIdentifier{ID: genTestID(t, id.UnitIDPrefix)}},
		},
	}
}

func ptrString(s string) *string {
	return &s
}

func (suite *ProductionRunBulkCreateTestSuite) expectResponsibleUserResolved(accountID, inputID, resolvedID string) {
	suite.accountUserRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), accountID, inputID).
		Return(resolvedID, nil).
		Times(1)
}

// expectBulkRefsResolved stubs the item and unit lookups so every item and unit in the
// given runs resolves successfully. The fixtures reference items by SKU and units by
// ID, so only FetchItemsBySKU and the unit GetByIDs are exercised (an item id-identifier would
// hit ItemRepo.GetByIDs, and a unit name/abbreviation identifier FindByAbbreviationsOrNames).
func (suite *ProductionRunBulkCreateTestSuite) expectBulkRefsResolved(accountID string, runs []domain.BulkCreateProductionRunParams) {
	itemRows := make([]domain.ItemSKUInfo, 0)
	seenSKU := map[string]struct{}{}
	seenUnit := map[string]struct{}{}
	units := make([]*domain.Unit, 0)

	collectUnit := func(identifier domain.UnitIdentifier) {
		if identifier.ID == "" {
			return
		}
		if _, ok := seenUnit[identifier.ID]; !ok {
			seenUnit[identifier.ID] = struct{}{}
			units = append(units, &domain.Unit{ID: identifier.ID, UnitDimensionCode: "count"})
		}
	}
	for _, run := range runs {
		for _, batch := range run.Batches {
			if sku := batch.Item.SKU; sku != "" {
				if _, ok := seenSKU[sku]; !ok {
					seenSKU[sku] = struct{}{}
					itemRows = append(itemRows, domain.ItemSKUInfo{
						SKU:    sku,
						ItemID: genTestID(suite.T(), id.ItemIDPrefix),
					})
				}
			}
			collectUnit(batch.QuantityUnit)
			if batch.SecondsUnit != nil {
				collectUnit(*batch.SecondsUnit)
			}
			if batch.WasteUnit != nil {
				collectUnit(*batch.WasteUnit)
			}
		}
	}

	suite.itemRepo.EXPECT().
		FetchItemsBySKU(gomock.Any(), accountID, gomock.Any()).
		Return(itemRows, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		GetByIDs(gomock.Any(), accountID, gomock.Any()).
		Return(units, nil).
		Times(1)
}

// expectIdempotencyStarted drives the request down the executing branch, which is
// also the only branch that raises a job: it resolves the acting user for the job's
// created_by and asks the job service to record the resolved payload. Both are
// stubbed permissively — the actor the identity happens to carry is not what these
// rows are about, and TestBulkCreateProductionRuns_RecordsJob pins the job itself.
func (suite *ProductionRunBulkCreateTestSuite) expectIdempotencyStarted(typeID string) {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        typeID,
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
	suite.accountUserRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), gomock.Any(), suite.actorID).
		Return(genTestID(suite.T(), id.AccountUserIDPrefix), nil).
		AnyTimes()
	suite.jobSvc.EXPECT().
		CreateJob(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
			// Echo the accept-time results onto the returned job, as the real CreateJob
			// persists then reads back: a create's pre-generated ids ride home on it.
			return &domain.Job{ID: genTestID(suite.T(), id.JobIDPrefix), Results: params.Results}, nil
		}).
		AnyTimes()
}

func (suite *ProductionRunBulkCreateTestSuite) bulkCreate(ctx context.Context, runs ...domain.BulkCreateProductionRunParams) (*domain.Job, *apierror.APIError) {
	return suite.productionRunSvc.BulkCreateProductionRuns(ctx, domain.BulkCreateProductionRunsParams{ProductionRuns: runs})
}

// --- BulkCreateProductionRuns ---

// Identity guards cannot be forged through the gateway, so they are verified here.
func (suite *ProductionRunBulkCreateTestSuite) TestBulkCreateProductionRuns_IdentityGuards() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	customRole := string(constants.RoleTypeCustom)
	identityCtx := func(relation types.IdentityRelationType, permissions map[string]bool) context.Context {
		return appctx.WithIdentity(context.Background(), &types.Identity{
			Type:   types.IdentityActorTypeUser,
			Target: &types.IdentityTarget{AccountID: accountID},
			Actor: &types.IdentityActor{
				RelationType: relation,
				ID:           genTestID(suite.T(), id.UserIDPrefix),
				AccountID:    &accountID,
				RoleType:     &customRole,
				Permissions:  permissions,
			},
		})
	}

	cases := []struct {
		name string
		ctx  context.Context
		code apierror.ErrorCode
	}{
		{"missing identity", context.Background(), apierror.ErrorCodeInternalError},
		{"missing target account", internalProductionRunCtx(suite.T(), "", suite.actorID), apierror.ErrorCodeInvalidCredentials},
		{"external actor", identityCtx(types.IdentityRelationTypeCustomer, map[string]bool{"production_runs:create": true}), apierror.ErrorCodeInsufficientPerms},
		{"missing create permission", identityCtx(types.IdentityRelationTypeInternal, map[string]bool{}), apierror.ErrorCodeInsufficientPerms},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			result, err := suite.bulkCreate(tc.ctx, validBulkCreateProductionRunParams(suite.T()))

			suite.Nil(result)
			suite.NotNil(err)
			suite.Equal(tc.code, err.Code)
		})
	}
}

// Structural validation runs before any repository access: batch count and the
// decimal parse of every quantity/seconds/waste value, each with a row-indexed
// param. A seconds or waste value without its unit carries no information and is
// ignored rather than rejected.
func (suite *ProductionRunBulkCreateTestSuite) TestBulkCreateProductionRuns_StructuralValidation() {
	ctx := internalProductionRunCtx(suite.T(), genTestID(suite.T(), id.AccountIDPrefix), suite.actorID)

	cases := []struct {
		name   string
		mutate func(run *domain.BulkCreateProductionRunParams)
		param  string
	}{
		{"no batches", func(run *domain.BulkCreateProductionRunParams) {
			run.Batches = nil
		}, "production_runs[0].batches"},
		{"non-decimal quantity", func(run *domain.BulkCreateProductionRunParams) {
			run.Batches[0].QuantityValue = "lots"
		}, "production_runs[0].batches[0].quantity_value"},
		{"non-decimal seconds", func(run *domain.BulkCreateProductionRunParams) {
			run.Batches[0].SecondsValue = ptrString("later")
			run.Batches[0].SecondsUnit = &domain.UnitIdentifier{ID: genTestID(suite.T(), id.UnitIDPrefix)}
		}, "production_runs[0].batches[0].seconds_value"},
		{"non-decimal waste", func(run *domain.BulkCreateProductionRunParams) {
			run.Batches[0].WasteValue = ptrString("scrap")
			run.Batches[0].WasteUnit = &domain.UnitIdentifier{ID: genTestID(suite.T(), id.UnitIDPrefix)}
		}, "production_runs[0].batches[0].waste_value"},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			run := validBulkCreateProductionRunParams(suite.T())
			tc.mutate(&run)

			result, err := suite.bulkCreate(ctx, run)

			suite.Nil(result)
			suite.NotNil(err)
			suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
			suite.Equal(tc.param, err.Param)
		})
	}

	suite.Run("empty", func() {
		result, err := suite.bulkCreate(ctx)
		suite.Nil(result)
		suite.NotNil(err)
		suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	})

	suite.Run("too many", func() {
		runs := make([]domain.BulkCreateProductionRunParams, 1001)
		for i := range runs {
			runs[i] = validBulkCreateProductionRunParams(suite.T())
		}
		result, err := suite.bulkCreate(ctx, runs...)
		suite.Nil(result)
		suite.NotNil(err)
		suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	})
}

// Repository and mediator failures cannot be forced through e2e; each row fails a
// different dependency and expects the error to surface instead of a partial write.
func (suite *ProductionRunBulkCreateTestSuite) TestBulkCreateProductionRuns_DependencyErrorsPropagated() {
	depErr := apierror.NewValidationError("dependency failed")

	cases := []struct {
		name  string
		setup func(accountID string, run domain.BulkCreateProductionRunParams)
		code  apierror.ErrorCode
	}{
		// These lookups all run inside Resolve, which follows the key claim, so each case stubs the key as started.
		{"responsible user lookup", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.accountUserRepo.EXPECT().
				ResolveAccountUserID(gomock.Any(), accountID, run.ResponsibleUserID).
				Return("", apierror.NewResourceNotFoundError("not found")).
				Times(1)
		}, apierror.ErrorCodeValidationFailed},
		{"item repo", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.expectResponsibleUserResolved(accountID, run.ResponsibleUserID, genTestID(suite.T(), id.AccountUserIDPrefix))
			suite.itemRepo.EXPECT().
				FetchItemsBySKU(gomock.Any(), accountID, gomock.Any()).
				Return(nil, depErr).
				Times(1)
		}, apierror.ErrorCodeValidationFailed},
		{"unit repo", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.expectResponsibleUserResolved(accountID, run.ResponsibleUserID, genTestID(suite.T(), id.AccountUserIDPrefix))
			suite.itemRepo.EXPECT().
				FetchItemsBySKU(gomock.Any(), accountID, gomock.Any()).
				Return([]domain.ItemSKUInfo{{SKU: run.Batches[0].Item.SKU, ItemID: genTestID(suite.T(), id.ItemIDPrefix)}}, nil).
				Times(1)
			suite.unitRepo.EXPECT().
				GetByIDs(gomock.Any(), accountID, gomock.Any()).
				Return(nil, depErr).
				Times(1)
		}, apierror.ErrorCodeValidationFailed},
		{"step repo", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.expectResponsibleUserResolved(accountID, run.ResponsibleUserID, genTestID(suite.T(), id.AccountUserIDPrefix))
			suite.expectBulkRefsResolved(accountID, []domain.BulkCreateProductionRunParams{run})
			suite.expectStationResolved(accountID, run.Batches[0].ScanningStation)
			suite.productionStepQueryRepo.EXPECT().
				IsInAccount(gomock.Any(), accountID, *run.Batches[0].ProductionStepID).
				Return(false, depErr).
				Times(1)
		}, apierror.ErrorCodeValidationFailed},
		// Station lookup is batched before the row-by-row pass, so a failure here
		// surfaces before the production-step check ever runs.
		{"station repo", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.expectResponsibleUserResolved(accountID, run.ResponsibleUserID, genTestID(suite.T(), id.AccountUserIDPrefix))
			suite.expectBulkRefsResolved(accountID, []domain.BulkCreateProductionRunParams{run})
			suite.scanningStationRepo.EXPECT().
				GetByIDs(gomock.Any(), accountID, gomock.Any()).
				Return(nil, depErr).
				Times(1)
		}, apierror.ErrorCodeValidationFailed},
		// Claiming the key comes first, so nothing is resolved before it fails.
		{"idempotency upsert", func(_ string, _ domain.BulkCreateProductionRunParams) {
			suite.idempotencyMed.EXPECT().
				UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
				Return(nil, depErr).
				Times(1)
		}, apierror.ErrorCodeValidationFailed},
		{"outbox insert", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.outboxRepo = &failingOutboxRepo{}
			suite.expectFullyResolved(accountID, run)
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.expectCacheErrorResponsePassthrough()
		}, apierror.ErrorCodeInternalError},
		{"cache success response", func(accountID string, run domain.BulkCreateProductionRunParams) {
			suite.expectFullyResolved(accountID, run)
			suite.expectIdempotencyStarted("idk_bulk_test")
			suite.idempotencyMed.EXPECT().
				CacheSuccessResponse(gomock.Any(), "idk_bulk_test", gomock.Any()).
				Return(depErr).
				Times(1)
			suite.expectCacheErrorResponsePassthrough()
		}, apierror.ErrorCodeValidationFailed},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.outboxRepo = &stubOutboxRepo{}
			accountID := genTestID(suite.T(), id.AccountIDPrefix)
			run := validBulkCreateProductionRunParams(suite.T())
			run.Batches[0].ProductionStepID = ptrString(genTestID(suite.T(), id.ProductionStepIDPrefix))
			run.Batches[0].ScanningStation = &domain.ObjectIdentifier{ID: genTestID(suite.T(), id.ScanningStationIDPrefix)}
			tc.setup(accountID, run)

			result, err := suite.bulkCreate(internalProductionRunCtx(suite.T(), accountID, suite.actorID), run)

			suite.Nil(result)
			suite.NotNil(err)
			suite.Equal(tc.code, err.Code)
		})
	}
}

// expectStationResolved stubs the batched scanning-station lookup so an ID-referenced
// station resolves successfully. A nil identifier means the batch names no station, so the
// resolver never queries.
func (suite *ProductionRunBulkCreateTestSuite) expectStationResolved(accountID string, identifier *domain.ObjectIdentifier) {
	if identifier == nil || identifier.ID == "" {
		return
	}
	suite.scanningStationRepo.EXPECT().
		GetByIDs(gomock.Any(), accountID, gomock.Any()).
		Return([]*domain.ScanningStation{{ID: identifier.ID}}, nil).
		Times(1)
}

// expectFullyResolved stubs successful resolution of every reference in the run,
// including its step and station.
func (suite *ProductionRunBulkCreateTestSuite) expectFullyResolved(accountID string, run domain.BulkCreateProductionRunParams) {
	suite.expectResponsibleUserResolved(accountID, run.ResponsibleUserID, genTestID(suite.T(), id.AccountUserIDPrefix))
	suite.expectBulkRefsResolved(accountID, []domain.BulkCreateProductionRunParams{run})
	suite.expectStationResolved(accountID, run.Batches[0].ScanningStation)
	if run.Batches[0].ProductionStepID != nil {
		suite.productionStepQueryRepo.EXPECT().
			IsInAccount(gomock.Any(), accountID, *run.Batches[0].ProductionStepID).
			Return(true, nil).
			Times(1)
	}
}

func (suite *ProductionRunBulkCreateTestSuite) expectCacheErrorResponsePassthrough() {
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_bulk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

// Recovery-point handling is internal to the service; e2e's replay test proves the
// behavior outside-in, these rows prove the cache and corruption branches.
func (suite *ProductionRunBulkCreateTestSuite) TestBulkCreateProductionRuns_IdempotencyRecoveryPoints() {
	code := 202
	cachedRunID := genTestID(suite.T(), id.ProductionRunIDPrefix)
	cached := &domain.Job{
		ID: genTestID(suite.T(), id.JobIDPrefix),
		Results: []domain.RowResult{
			runRowResult(0, cachedRunID, []string{genTestID(suite.T(), id.BatchIDPrefix)}),
		},
	}

	cases := []struct {
		name    string
		key     *domain.IdempotencyKey
		errCode apierror.ErrorCode // zero value means success expected
	}{
		{"finished replays cached response", &domain.IdempotencyKey{
			TypeID:        "idk_bulk_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(cached),
		}, ""},
		{"finished with corrupt cache", &domain.IdempotencyKey{
			TypeID:        "idk_bulk_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  nil,
		}, apierror.ErrorCodeInternalError},
		{"unexpected recovery point", &domain.IdempotencyKey{
			TypeID:        "idk_bulk_test",
			RecoveryPoint: "core:confused",
		}, apierror.ErrorCodeInternalError},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			accountID := genTestID(suite.T(), id.AccountIDPrefix)
			run := validBulkCreateProductionRunParams(suite.T())
			// No resolution stubbed: none of these branches resolves, and gomock fails the test if one tries.
			suite.idempotencyMed.EXPECT().
				UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
				Return(tc.key, nil).
				Times(1)

			result, err := suite.bulkCreate(internalProductionRunCtx(suite.T(), accountID, suite.actorID), run)

			if tc.errCode == "" {
				suite.Nil(err)
				suite.NotNil(result)
				suite.Equal(cached.ID, result.ID)
				suite.Equal(cachedRunID, result.Results[0].ID)
			} else {
				suite.Nil(result)
				suite.NotNil(err)
				suite.Equal(tc.errCode, err.Code)
			}
		})
	}
}

// The happy path folds in two behaviors e2e can't observe: a responsible user
// shared across runs is resolved once, and a seconds/waste value without its unit
// is ignored (the pair only travels when both halves are present).
func (suite *ProductionRunBulkCreateTestSuite) TestBulkCreateProductionRuns_Succeeds() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	runA := validBulkCreateProductionRunParams(suite.T())
	runA.Batches[0].SecondsValue = ptrString("not-a-decimal")
	runA.Batches[0].WasteValue = ptrString("also-not-a-decimal")
	runB := validBulkCreateProductionRunParams(suite.T())
	runB.ResponsibleUserID = runA.ResponsibleUserID
	runs := []domain.BulkCreateProductionRunParams{runA, runB}

	suite.expectResponsibleUserResolved(accountID, runA.ResponsibleUserID, genTestID(suite.T(), id.AccountUserIDPrefix))
	suite.expectBulkRefsResolved(accountID, runs)
	suite.expectIdempotencyStarted("idk_bulk_test")
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_bulk_test", gomock.Any()).
		Return(nil).
		Times(1)

	result, err := suite.bulkCreate(internalProductionRunCtx(suite.T(), accountID, suite.actorID), runs...)

	suite.Nil(err)
	suite.NotNil(result)
	suite.Len(result.Results, 2)
}

// The job is the record of the requested work, and the ID it comes back with is the
// only handle a caller has on an execution that has not happened yet — so that ID has
// to reach the acknowledgment. The payload handed to the job is what the consumer will
// execute, so it must carry the resolved references and the same pre-generated IDs the
// caller was acknowledged with: if the two ever diverged, the client would be told
// about rows that never get created.
func (suite *ProductionRunBulkCreateTestSuite) TestBulkCreateProductionRuns_RecordsJob() {
	accountID := genTestID(suite.T(), id.AccountIDPrefix)
	resolvedUserID := genTestID(suite.T(), id.AccountUserIDPrefix)
	creatorID := genTestID(suite.T(), id.AccountUserIDPrefix)
	jobID := genTestID(suite.T(), id.JobIDPrefix)
	run := validBulkCreateProductionRunParams(suite.T())

	suite.expectResponsibleUserResolved(accountID, run.ResponsibleUserID, resolvedUserID)
	// The acting user is resolved separately from the responsible user, and it is
	// the acting one the job is attributed to.
	suite.expectResponsibleUserResolved(accountID, suite.actorID, creatorID)
	suite.expectBulkRefsResolved(accountID, []domain.BulkCreateProductionRunParams{run})

	var recorded domain.CreateJobServiceParams
	suite.jobSvc.EXPECT().
		CreateJob(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
			recorded = params
			return &domain.Job{ID: jobID, Results: params.Results}, nil
		}).
		Times(1)

	suite.expectIdempotencyStarted("idk_bulk_test")
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_bulk_test", gomock.Any()).
		Return(nil).
		Times(1)

	result, err := suite.bulkCreate(internalProductionRunCtx(suite.T(), accountID, suite.actorID), run)

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal(constants.JobTypeBulkCreate, recorded.Type)
	suite.Equal(&creatorID, recorded.CreatedByID)
	suite.Equal(jobID, result.ID)

	var storedRuns []domain.BulkCreateProductionRunEventRun
	suite.NoError(json.Unmarshal(recorded.JobItems, &storedRuns))
	suite.Len(storedRuns, 1)
	suite.Equal(resolvedUserID, storedRuns[0].ResponsibleUserID)

	// The pre-generated ids the job carries in its results must be the same ids stored
	// in the payload the consumer will execute: the client is acknowledged with exactly
	// what will be created.
	jobRuns := result.Results
	suite.Equal(jobRuns[0].ID, storedRuns[0].ProductionRunID)
	suite.Equal(jobRuns[0].SubResources[0].ID, storedRuns[0].Batches[0].BatchID)
}

// --- ExecuteBulkCreateProductionRuns (consumer side) ---

// bulkCreateEventFixture pairs an account with the resolved runs a bulk create job
// stores. The job payload is just the runs (the account comes from the identity); this
// fixture keeps the two together so a test can stub the job load and the identity from
// one value.
type bulkCreateEventFixture struct {
	AccountID string
	Runs      []domain.BulkCreateProductionRunEventRun
}

func validBulkCreateEvent(t *testing.T) bulkCreateEventFixture {
	t.Helper()
	return bulkCreateEventFixture{
		AccountID: genTestID(t, id.AccountIDPrefix),
		Runs: []domain.BulkCreateProductionRunEventRun{
			{
				ProductionRunID:   genTestID(t, id.ProductionRunIDPrefix),
				ResponsibleUserID: genTestID(t, id.AccountUserIDPrefix),
				Batches: []domain.BulkCreateProductionRunEventBatch{
					{
						BatchID:        genTestID(t, id.BatchIDPrefix),
						ItemID:         genTestID(t, id.ItemIDPrefix),
						QuantityValue:  "100",
						QuantityUnitID: genTestID(t, id.UnitIDPrefix),
					},
				},
			},
		},
	}
}

// executeEvent invokes Execute the way the consumer does: the message carries only
// a job ID, so the payload is stubbed onto the job the service loads, and the
// caller's identity is restored onto the context — the audit publish inside requires
// it. The lifecycle marks are stubbed permissively here; the rows below assert what
// the writes do, and TestExecuteBulkCreateProductionRuns_JobLifecycle asserts the
// marks themselves.
func (suite *ProductionRunBulkCreateTestSuite) executeEvent(event bulkCreateEventFixture) *apierror.APIError {
	jobID := genTestID(suite.T(), id.JobIDPrefix)

	suite.expectJobLoaded(jobID, event, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(time.Now(), nil).AnyTimes()
	suite.jobSvc.EXPECT().CompleteJob(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	suite.jobSvc.EXPECT().FailJob(gomock.Any(), gomock.Any()).AnyTimes()

	return suite.executeJob(jobID, event.AccountID)
}

// executeJob invokes Execute with the job-ID-only message the consumer receives,
// against the identity the consumer restores from it.
func (suite *ProductionRunBulkCreateTestSuite) executeJob(jobID, accountID string) *apierror.APIError {
	return suite.productionRunSvc.ExecuteBulkCreateProductionRuns(
		internalProductionRunCtx(suite.T(), accountID, suite.actorID),
		domain.BulkOperationJobEvent{JobID: jobID},
	)
}

// Each row fails one stage of the write pipeline; earlier stages are stubbed to
// succeed. The corrupt-value row covers the decimal parse shared by quantity,
// seconds, and waste.
func (suite *ProductionRunBulkCreateTestSuite) TestExecuteBulkCreateProductionRuns_WriteErrorsPropagated() {
	writeErr := apierror.NewValidationError("write failed")

	// The batch's numbers are allocated once, ahead of every row stage.
	stubNumbers := func(event bulkCreateEventFixture) {
		suite.productionRunRepo.EXPECT().
			GetNextNumbers(gomock.Any(), event.AccountID, len(event.Runs)).
			Return([]string{"PR-1"}, nil).
			Times(1)
	}

	// Stages in pipeline order; a case at depth N stubs stages 0..N-1 as successful.
	type stage func(event bulkCreateEventFixture, failing bool)
	stages := []struct {
		name  string
		stage stage
	}{
		{"create run", func(event bulkCreateEventFixture, failing bool) {
			call := suite.productionRunRepo.EXPECT().
				Create(gomock.Any(), event.Runs[0].ProductionRunID, gomock.Any(), "PR-1").
				Times(1)
			if failing {
				call.Return(nil, writeErr)
			} else {
				call.Return(&domain.ProductionRun{
					ID:                event.Runs[0].ProductionRunID,
					Number:            "PR-1",
					AccountID:         event.AccountID,
					ResponsibleUserID: event.Runs[0].ResponsibleUserID,
				}, nil)
			}
		}},
		{"create batch", func(event bulkCreateEventFixture, failing bool) {
			call := suite.batchRepo.EXPECT().
				Create(gomock.Any(), event.Runs[0].Batches[0].BatchID, gomock.Any()).
				Times(1)
			if failing {
				call.Return(nil, writeErr)
			} else {
				call.Return(&domain.BaseBatch{ID: event.Runs[0].Batches[0].BatchID}, nil)
			}
		}},
		{"link batch to run", func(event bulkCreateEventFixture, failing bool) {
			call := suite.productionRunRepo.EXPECT().
				SetBatchProductionRunID(gomock.Any(), event.AccountID, event.Runs[0].Batches[0].BatchID, event.Runs[0].ProductionRunID).
				Times(1)
			if failing {
				call.Return(writeErr)
			} else {
				call.Return(nil)
			}
		}},
	}

	// executeCapturing runs the single-run event and returns what the job was completed
	// with. A per-row write failure is recorded, not propagated: the job completes.
	executeCapturing := func(event bulkCreateEventFixture) (jobLifecycleCalls, *apierror.APIError) {
		jobID := genTestID(suite.T(), id.JobIDPrefix)
		suite.expectJobLoaded(jobID, event, nil)
		var calls jobLifecycleCalls
		suite.captureJobLifecycle(jobID, &calls)
		return calls, suite.executeJob(jobID, event.AccountID)
	}

	// assertRunRecordedAsFailed checks the single run was recorded as a failed row (not
	// created) and the job still completed.
	assertRunRecordedAsFailed := func(calls jobLifecycleCalls, err *apierror.APIError) {
		suite.Nil(err, "a per-row write failure is recorded, not propagated")
		suite.Equal([]string{"start", "complete"}, calls.order)
		results := calls.completed[0].Results
		suite.Len(results, 1)
		suite.Equal(0, results[0].Index)
		suite.Equal(constants.JobResultStatusFailed, results[0].Status)
		suite.Empty(results[0].ID, "the failed run is not reported as created")
	}

	for depth, failingStage := range stages {
		suite.Run(failingStage.name, func() {
			event := validBulkCreateEvent(suite.T())
			stubNumbers(event)
			for i := 0; i < depth; i++ {
				stages[i].stage(event, false)
			}
			failingStage.stage(event, true)

			calls, err := executeCapturing(event)
			assertRunRecordedAsFailed(calls, err)
		})
	}

	suite.Run("corrupt quantity value", func() {
		event := validBulkCreateEvent(suite.T())
		event.Runs[0].Batches[0].QuantityValue = "many"
		stubNumbers(event)
		stages[0].stage(event, false)

		calls, err := executeCapturing(event)
		assertRunRecordedAsFailed(calls, err)
	})

	// The audit event for each created run publishes through the outbox inside that run's
	// savepoint; an outbox failure rolls back only that run and records it as failed.
	suite.Run("audit publish", func() {
		suite.outboxRepo = &failingOutboxRepo{}
		defer func() { suite.outboxRepo = &stubOutboxRepo{} }()
		event := validBulkCreateEvent(suite.T())
		stubNumbers(event)
		stages[0].stage(event, false)

		calls, err := executeCapturing(event)
		assertRunRecordedAsFailed(calls, err)
	})
}

// expectRunWritesSucceed stubs the whole write pipeline for a single-run,
// single-batch event so a test can be about the job marks rather than the writes.
func (suite *ProductionRunBulkCreateTestSuite) expectRunWritesSucceed(event bulkCreateEventFixture) {
	run := event.Runs[0]
	suite.productionRunRepo.EXPECT().
		GetNextNumbers(gomock.Any(), event.AccountID, len(event.Runs)).
		Return([]string{"PR-1"}, nil).
		Times(1)
	suite.productionRunRepo.EXPECT().
		Create(gomock.Any(), run.ProductionRunID, gomock.Any(), "PR-1").
		Return(&domain.ProductionRun{ID: run.ProductionRunID, Number: "PR-1", AccountID: event.AccountID}, nil).
		Times(1)
	suite.batchRepo.EXPECT().
		Create(gomock.Any(), run.Batches[0].BatchID, gomock.Any()).
		Return(&domain.BaseBatch{ID: run.Batches[0].BatchID}, nil).
		Times(1)
	suite.productionRunRepo.EXPECT().
		SetBatchProductionRunID(gomock.Any(), event.AccountID, run.Batches[0].BatchID, run.ProductionRunID).
		Return(nil).
		Times(1)
}

// expectJobLoaded stubs the job the consumer loads, carrying the given payload.
func (suite *ProductionRunBulkCreateTestSuite) expectJobLoaded(jobID string, event bulkCreateEventFixture, job *domain.Job) {
	accountID := event.AccountID
	if job == nil {
		job = &domain.Job{ID: jobID, Type: constants.JobTypeBulkCreate, AccountID: &accountID}
	}
	job.JobItems = marshalJSON(event.Runs)
	suite.jobSvc.EXPECT().
		GetJobForExecution(gomock.Any(), jobID).
		Return(job, nil).
		Times(1)
}

// jobLifecycleCalls is the sequence of lifecycle calls Execute made against a job.
// Order is recorded because when a mark happens is as much of the contract as that
// it happened: started has to precede the writes and settle has to follow them.
type jobLifecycleCalls struct {
	order     []string
	completed []domain.CompleteJobParams
	failed    []domain.FailJobParams
}

// captureJobLifecycle records what Execute asks of the given job. It matches on the
// job ID rather than accepting anything: subtests share one gomock controller, so a
// catch-all registered by one would swallow its siblings' calls.
func (suite *ProductionRunBulkCreateTestSuite) captureJobLifecycle(jobID string, calls *jobLifecycleCalls) {
	suite.jobSvc.EXPECT().
		StartJob(gomock.Any(), gomock.Cond(func(p domain.StartJobParams) bool { return p.JobID == jobID })).
		DoAndReturn(func(context.Context, domain.StartJobParams) (time.Time, *apierror.APIError) {
			calls.order = append(calls.order, "start")
			return time.Now(), nil
		}).
		AnyTimes()
	suite.jobSvc.EXPECT().
		CompleteJob(gomock.Any(), gomock.Cond(func(p domain.CompleteJobParams) bool { return p.JobID == jobID })).
		DoAndReturn(func(_ context.Context, params domain.CompleteJobParams) *apierror.APIError {
			calls.order = append(calls.order, "complete")
			calls.completed = append(calls.completed, params)
			return nil
		}).
		AnyTimes()
	suite.jobSvc.EXPECT().
		FailJob(gomock.Any(), gomock.Cond(func(p domain.FailJobParams) bool { return p.JobID == jobID })).
		Do(func(_ context.Context, params domain.FailJobParams) {
			calls.order = append(calls.order, "fail")
			calls.failed = append(calls.failed, params)
		}).
		AnyTimes()
}

// The job marks are the only thing a client polling the job can see, so what Execute
// records — and when — is the observable contract of the async path.
func (suite *ProductionRunBulkCreateTestSuite) TestExecuteBulkCreateProductionRuns_JobLifecycle() {
	// Started is marked before the writes and completion after them, and the
	// completion carries the rows that were created so the caller can see what
	// it got without querying each one.
	suite.Run("success marks started then completed with results", func() {
		event := validBulkCreateEvent(suite.T())
		jobID := genTestID(suite.T(), id.JobIDPrefix)
		suite.expectJobLoaded(jobID, event, nil)
		suite.expectRunWritesSucceed(event)

		var calls jobLifecycleCalls
		suite.captureJobLifecycle(jobID, &calls)

		err := suite.executeJob(jobID, event.AccountID)

		suite.Nil(err)
		suite.Equal([]string{"start", "complete"}, calls.order)

		results := calls.completed[0].Results
		suite.Len(results, 1)
		suite.Equal(event.Runs[0].ProductionRunID, results[0].ID)
		suite.Equal(constants.ObjectTypeProductionRun, results[0].ResourceType)
		suite.Equal(
			[]domain.SubResourceRef{{ResourceType: constants.ObjectTypeBatch, ID: event.Runs[0].Batches[0].BatchID}},
			results[0].SubResources,
		)
	})

	// A row that fails to write is recorded in the job's errors and the job still
	// completes (partial success): the failure is visible to a client polling the job,
	// and does not roll back the rows that did write.
	suite.Run("write failure is recorded and the job completes", func() {
		event := validBulkCreateEvent(suite.T())
		jobID := genTestID(suite.T(), id.JobIDPrefix)
		writeErr := apierror.NewValidationError("write failed")
		suite.expectJobLoaded(jobID, event, nil)
		suite.productionRunRepo.EXPECT().
			GetNextNumbers(gomock.Any(), event.AccountID, len(event.Runs)).
			Return([]string{"PR-1"}, nil).
			Times(1)
		suite.productionRunRepo.EXPECT().
			Create(gomock.Any(), event.Runs[0].ProductionRunID, gomock.Any(), "PR-1").
			Return(nil, writeErr).
			Times(1)

		var calls jobLifecycleCalls
		suite.captureJobLifecycle(jobID, &calls)

		err := suite.executeJob(jobID, event.AccountID)

		suite.Nil(err)
		suite.Equal([]string{"start", "complete"}, calls.order)

		results := calls.completed[0].Results
		suite.Len(results, 1)
		suite.Equal(constants.JobResultStatusFailed, results[0].Status)
		suite.Require().NotNil(results[0].Error)
		suite.Equal("write failed", results[0].Error.Message)
	})

	// The batch's numbers come from one locked read, so an allocation failure leaves no
	// row writable: the whole batch rolls back and the job fails rather than completing.
	suite.Run("number allocation failure fails the job", func() {
		event := validBulkCreateEvent(suite.T())
		jobID := genTestID(suite.T(), id.JobIDPrefix)
		allocErr := apierror.NewValidationError("allocation failed")
		suite.expectJobLoaded(jobID, event, nil)
		suite.productionRunRepo.EXPECT().
			GetNextNumbers(gomock.Any(), event.AccountID, len(event.Runs)).
			Return(nil, allocErr).
			Times(1)

		var calls jobLifecycleCalls
		suite.captureJobLifecycle(jobID, &calls)

		err := suite.executeJob(jobID, event.AccountID)

		suite.NotNil(err)
		suite.Equal([]string{"start", "fail"}, calls.order)
	})

	// A redelivery of a job that already settled must not create its runs a second
	// time: no write is stubbed here, so any attempt to write fails the test.
	suite.Run("settled job is not executed again", func() {
		event := validBulkCreateEvent(suite.T())
		jobID := genTestID(suite.T(), id.JobIDPrefix)
		accountID := event.AccountID
		completedAt := time.Now().UTC()
		suite.expectJobLoaded(jobID, event, &domain.Job{
			ID:          jobID,
			Type:        constants.JobTypeBulkCreate,
			AccountID:   &accountID,
			CompletedAt: &completedAt,
		})

		err := suite.executeJob(jobID, accountID)

		suite.Nil(err)
	})

	// A failed job is deliberately not settled: the inbox redelivers on error, and
	// that retry has to be allowed to run the writes.
	suite.Run("previously failed job is retried", func() {
		event := validBulkCreateEvent(suite.T())
		jobID := genTestID(suite.T(), id.JobIDPrefix)
		accountID := event.AccountID
		failedAt := time.Now().UTC()
		suite.expectJobLoaded(jobID, event, &domain.Job{
			ID:        jobID,
			Type:      constants.JobTypeBulkCreate,
			AccountID: &accountID,
			StartedAt: &failedAt,
			FailedAt:  &failedAt,
		})
		suite.expectRunWritesSucceed(event)

		var calls jobLifecycleCalls
		suite.captureJobLifecycle(jobID, &calls)

		err := suite.executeJob(jobID, accountID)

		suite.Nil(err)
		suite.Equal([]string{"start", "complete"}, calls.order)
	})
}

// Full-fidelity success: two runs with sequential numbers, one carrying every batch
// field and one carrying a value-without-unit pair that Execute must skip.
func (suite *ProductionRunBulkCreateTestSuite) TestExecuteBulkCreateProductionRuns_Succeeds() {
	event := validBulkCreateEvent(suite.T())
	event.Runs[0].Batches[0].SecondsValue = ptrString("60")
	event.Runs[0].Batches[0].SecondsUnitID = ptrString(genTestID(suite.T(), id.UnitIDPrefix))
	event.Runs[0].Batches[0].WasteValue = ptrString("1")
	event.Runs[0].Batches[0].WasteUnitID = ptrString(genTestID(suite.T(), id.UnitIDPrefix))
	event.Runs[0].Batches[0].ProductionStepID = ptrString(genTestID(suite.T(), id.ProductionStepIDPrefix))
	event.Runs[0].Batches[0].ScanningStationID = ptrString(genTestID(suite.T(), id.ScanningStationIDPrefix))
	event.Runs = append(event.Runs, domain.BulkCreateProductionRunEventRun{
		ProductionRunID:   genTestID(suite.T(), id.ProductionRunIDPrefix),
		ResponsibleUserID: genTestID(suite.T(), id.AccountUserIDPrefix),
		Batches: []domain.BulkCreateProductionRunEventBatch{
			{
				BatchID:        genTestID(suite.T(), id.BatchIDPrefix),
				ItemID:         genTestID(suite.T(), id.ItemIDPrefix),
				QuantityValue:  "25",
				QuantityUnitID: genTestID(suite.T(), id.UnitIDPrefix),
				// Value without a unit: enqueue copies the value but omits the
				// unit, so Execute must skip the pair, not parse it.
				SecondsValue: ptrString("not-a-decimal"),
			},
		},
	})

	numbers := []string{"PR-1", "PR-2"}
	suite.productionRunRepo.EXPECT().
		GetNextNumbers(gomock.Any(), event.AccountID, len(event.Runs)).
		Return(numbers, nil).
		Times(1)
	for i, run := range event.Runs {
		suite.productionRunRepo.EXPECT().
			Create(gomock.Any(), run.ProductionRunID, gomock.Any(), numbers[i]).
			Return(&domain.ProductionRun{
				ID:                run.ProductionRunID,
				Number:            numbers[i],
				AccountID:         event.AccountID,
				ResponsibleUserID: run.ResponsibleUserID,
			}, nil).
			Times(1)
		for _, batch := range run.Batches {
			suite.batchRepo.EXPECT().
				Create(gomock.Any(), batch.BatchID, gomock.Any()).
				Return(&domain.BaseBatch{ID: batch.BatchID}, nil).
				Times(1)
			suite.productionRunRepo.EXPECT().
				SetBatchProductionRunID(gomock.Any(), event.AccountID, batch.BatchID, run.ProductionRunID).
				Return(nil).
				Times(1)
		}
	}

	err := suite.executeEvent(event)

	suite.Nil(err)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// reopens the export so assertions read what a spreadsheet would
func (suite *ProductionRunBulkCreateTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Production Runs")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *ProductionRunBulkCreateTestSuite) buildExport(ctx context.Context, params domain.ExportProductionRunsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.productionRunSvc.(*productionRunSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

// the run's own columns sit on its first batch's row and are blank on the rest
func (suite *ProductionRunBulkCreateTestSuite) TestExportProductionRuns_ListsBatchesOnePerRow() {
	ctx := internalProductionRunCtx(suite.T(), "ac_test123", "acu_1")
	started := time.Date(2026, time.July, 1, 9, 0, 0, 0, time.UTC)
	scanned := time.Date(2026, time.July, 2, 14, 30, 0, 0, time.UTC)
	orderID := "so_1"

	suite.productionRunRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.ProductionRunExport{
		{
			ID:                  "prun_1",
			Number:              "R-100",
			ResponsibleUserName: "Ada",
			StartedAt:           &started,
			OrderID:             &orderID,
			Batches: []domain.ProductionRunExportBatch{
				{ItemSKU: "SKU-1", QuantityValue: "10", QuantityUnit: "kg", MachineNames: []string{"Loom 1", "Loom 2"}, ScannedAt: &scanned},
				{ItemSKU: "SKU-2", QuantityValue: "5", QuantityUnit: "kg"},
			},
		},
		{ID: "prun_2", Number: "R-101", ResponsibleUserName: "Grace"},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportProductionRunsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount, "two runs, even though they span three sheet rows")

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 4)
	suite.Equal([]string{
		"ID", "Number", "Responsible User", "Started At", "Completed At", "Order ID",
		"Batch Item", "Batch Quantity", "Batch Unit", "Batch Department", "Batch Machines", "Batch Scanned At",
	}, rows[0])
	suite.Equal([]string{
		"prun_1", "R-100", "Ada", "07-01-2026", "", "so_1",
		"SKU-1", "10", "kg", "", "Loom 1; Loom 2", "07-02-2026",
	}, rows[1])
	// The continuation row carries only the batch; the run's columns are blank.
	suite.Equal([]string{"", "", "", "", "", "", "SKU-2", "5", "kg"}, rows[2])
	// A run with no batches still gets a row.
	suite.Equal([]string{"prun_2", "R-101", "Grace"}, rows[3])
}

func (suite *ProductionRunBulkCreateTestSuite) TestExportProductionRuns_ScopesToTheIdentitysAccount() {
	ctx := internalProductionRunCtx(suite.T(), "ac_owner", "acu_1")
	suite.productionRunRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportProductionRunsParams) ([]*domain.ProductionRunExport, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportProductionRunsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}
