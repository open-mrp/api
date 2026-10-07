package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

const (
	undoAccountID = "acct_test"
	undoStepID    = "prst_1"
	undoStationID = "scst_1"
	undoRunID     = "prun_1"
)

// recordingOutboxRepo keeps what the service enqueued so tests can assert on the undo command
// alongside the audit events that share the outbox.
type recordingOutboxRepo struct {
	messages []messaging.OutboxMessageInput
}

func (r *recordingOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	r.messages = append(r.messages, input)
	return int64(len(r.messages)), nil
}

func (r *recordingOutboxRepo) undoEvents() []domain.UndoBatchScanEvent {
	var events []domain.UndoBatchScanEvent
	for _, msg := range r.messages {
		if msg.MessageType != string(contracts.CoreCmdUndoBatchScan) {
			continue
		}
		var evt domain.UndoBatchScanEvent
		if err := json.Unmarshal(msg.Payload.Data, &evt); err != nil {
			continue
		}
		events = append(events, evt)
	}
	return events
}

type undoStubTxManager struct {
	factory domain.RepoFactory
}

func (m *undoStubTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.factory)
}

func (m *undoStubTxManager) WithTxSavepoint(ctx context.Context, fn func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.factory, passthroughSavepoint{})
}

type BatchUndoTestSuite struct {
	suite.Suite
	ctrl              *gomock.Controller
	svc               domain.BatchSvc
	batchRepo         *repositorymock.MockBatchRepo
	inventoryMutRepo  *repositorymock.MockInventoryMutationRepo
	stepRepo          *repositorymock.MockProductionStepQueryRepo
	runRepo           *repositorymock.MockProductionRunQueryRepo
	orderRepo         *repositorymock.MockOrderQueryRepo
	deletedRecordRepo *repositorymock.MockDeletedRecordRepo
	runActivity       *mediatormock.MockProductionRunActivityMed
	outbox            *recordingOutboxRepo
}

func (s *BatchUndoTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.batchRepo = repositorymock.NewMockBatchRepo(s.ctrl)
	s.inventoryMutRepo = repositorymock.NewMockInventoryMutationRepo(s.ctrl)
	s.stepRepo = repositorymock.NewMockProductionStepQueryRepo(s.ctrl)
	s.runRepo = repositorymock.NewMockProductionRunQueryRepo(s.ctrl)
	s.orderRepo = repositorymock.NewMockOrderQueryRepo(s.ctrl)
	s.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(s.ctrl)
	s.runActivity = mediatormock.NewMockProductionRunActivityMed(s.ctrl)
	s.outbox = &recordingOutboxRepo{}

	repoFactory := factorymock.NewMockRepoFactory(s.ctrl)
	repoFactory.EXPECT().NewBatchRepo().Return(s.batchRepo).AnyTimes()
	repoFactory.EXPECT().NewInventoryMutationRepo().Return(s.inventoryMutRepo).AnyTimes()
	repoFactory.EXPECT().NewProductionStepQueryRepo().Return(s.stepRepo).AnyTimes()
	repoFactory.EXPECT().NewProductionRunQueryRepo().Return(s.runRepo).AnyTimes()
	repoFactory.EXPECT().NewOrderQueryRepo().Return(s.orderRepo).AnyTimes()
	repoFactory.EXPECT().NewDeletedRecordRepo().Return(s.deletedRecordRepo).AnyTimes()
	repoFactory.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()

	mediatorFactory := factorymock.NewMockMediatorFactory(s.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{ProductionRunActivity: s.runActivity}).AnyTimes()

	s.svc = NewBatchSvc(&BatchSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &undoStubTxManager{factory: repoFactory},
	})
}

func (s *BatchUndoTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestBatchUndoTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BatchUndoTestSuite))
}

func undoIdentityCtx() context.Context {
	accountID := undoAccountID
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "acus_test",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions:  map[string]bool{},
		},
	})
}

type batchOption func(*domain.Batch)

func scannedBatch(id string, opts ...batchOption) *domain.Batch {
	scannedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	batch := &domain.Batch{
		ID:              id,
		Item:            domain.LightItem{ID: "it_product"},
		Quantity:        domain.BatchQuantity{ID: "qu_1", Measure: decimal.NewFromInt(10), Unit: domain.LightUnit{ID: "each"}},
		ScannedAt:       &scannedAt,
		ScanningStation: &domain.LightScanningStation{ID: undoStationID},
		ProductionStep:  &domain.LightProductionStep{ID: undoStepID},
	}
	for _, opt := range opts {
		opt(batch)
	}
	return batch
}

func inRun() batchOption {
	return func(b *domain.Batch) { b.ProductionRun = &domain.LightProductionRun{ID: undoRunID} }
}

func unscanned() batchOption {
	return func(b *domain.Batch) {
		b.ScannedAt = nil
		b.ProductionStep = nil
		b.ScanningStation = nil
	}
}

// expectNoShortfall makes the lineage read report a run with no scrap, which is the common case: the
// undo then carries no reservation snapshot.
func (s *BatchUndoTestSuite) expectNoShortfall(batchID string) {
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), batchID).
		Return(&domain.LineageShortfall{Seconds: decimal.Zero, Waste: decimal.Zero}, nil)
}

// expectHeld makes the undo's row locks find each batch as it was read.
func (s *BatchUndoTestSuite) expectHeld(batches ...*domain.Batch) {
	for _, b := range batches {
		s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, b.ID).Return(b.ScannedAt, true, nil)
	}
}

// expectScanUndoPlanned makes the reads an undo of a scanned batch plans with before holding its rows.
func (s *BatchUndoTestSuite) expectScanUndoPlanned(batch *domain.Batch) {
	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, batch.ID).Return(batch, nil)
	s.batchRepo.EXPECT().FindInputBatchIDs(gomock.Any(), batch.ID).Return(nil, nil)
	s.expectNoShortfall(batch.ID)
}

func (s *BatchUndoTestSuite) TestRefusesWhenALaterScanConsumedTheBatch() {
	batch := scannedBatch("bt_1")
	s.expectScanUndoPlanned(batch)
	s.expectHeld(batch)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), batch.ID).Return(int64(1), nil)

	_, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().NotNil(apiErr)
	s.Contains(apiErr.PublicMessage, "later scan")
	s.Empty(s.outbox.undoEvents())
}

func (s *BatchUndoTestSuite) TestRefusesWhenTheOutputHasAlreadyBeenDrawnOn() {
	batch := scannedBatch("bt_1")
	s.expectScanUndoPlanned(batch)
	s.expectHeld(batch)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), batch.ID).Return(int64(0), nil)
	s.inventoryMutRepo.EXPECT().CountAllocatedReceiptsForBatch(gomock.Any(), undoAccountID, batch.ID).Return(int64(2), nil)

	_, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().NotNil(apiErr)
	s.Contains(apiErr.PublicMessage, "already been used")
	s.Empty(s.outbox.undoEvents())
}

func (s *BatchUndoTestSuite) TestDeletesAPlannedBatchWithoutQueueingAnUndo() {
	batch := scannedBatch("bt_1", unscanned(), inRun())
	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, batch.ID).Return(batch, nil)
	s.expectHeld(batch)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), batch.ID).Return(int64(0), nil)
	s.inventoryMutRepo.EXPECT().CountAllocatedReceiptsForBatch(gomock.Any(), undoAccountID, batch.ID).Return(int64(0), nil)
	s.deletedRecordRepo.EXPECT().CreateInAccount(gomock.Any(), gomock.Any(), batch.ID, undoAccountID, gomock.Any()).Return(nil)
	s.batchRepo.EXPECT().Delete(gomock.Any(), undoAccountID, batch.ID).Return(&domain.BaseBatch{ID: batch.ID}, nil)
	s.runActivity.EXPECT().NotifyBatchDeleted(gomock.Any(), gomock.Any(), undoAccountID, batch).Return(nil)
	s.runRepo.EXPECT().CloseIfAllBatchesScannedOrDeleted(gomock.Any(), undoAccountID, undoRunID).Return(nil)

	result, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().Nil(apiErr)
	s.Equal(batch.ID, result.ID)
	s.Empty(s.outbox.undoEvents())
}

func (s *BatchUndoTestSuite) TestUnscansAnInitBatchAndReopensItsRun() {
	batch := scannedBatch("bt_1", inRun())
	s.expectScanUndoPlanned(batch)
	s.expectHeld(batch)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), batch.ID).Return(int64(0), nil)
	s.inventoryMutRepo.EXPECT().CountAllocatedReceiptsForBatch(gomock.Any(), undoAccountID, batch.ID).Return(int64(0), nil)
	s.batchRepo.EXPECT().Unscan(gomock.Any(), undoAccountID, batch.ID).Return(&domain.BaseBatch{ID: batch.ID}, nil)
	s.runRepo.EXPECT().Reopen(gomock.Any(), undoAccountID, undoRunID).Return(nil)

	result, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().Nil(apiErr)
	s.Equal(batch.ID, result.ID)

	events := s.outbox.undoEvents()
	s.Require().Len(events, 1)
	s.Equal(batch.ID, events[0].BatchID)
	s.Equal(undoStationID, events[0].ScanningStationID)
	s.Equal("acus_test", events[0].ResponsibleUserID)
}

func (s *BatchUndoTestSuite) TestDeletesABatchAScanCreatedAndReleasesItsInputs() {
	batch := scannedBatch("bt_1")
	inputs := []string{"bt_b", "bt_a"}

	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, batch.ID).Return(batch, nil)
	s.batchRepo.EXPECT().FindInputBatchIDs(gomock.Any(), batch.ID).Return(inputs, nil)
	s.expectNoShortfall(batch.ID)
	s.stepRepo.EXPECT().FindProducedUnit(gomock.Any(), undoAccountID, undoStepID).Return(&domain.LightUnit{ID: "each"}, nil)
	// The batch and its inputs are held in id order, so two undos or scans over them queue rather than deadlock.
	gomock.InOrder(
		s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, "bt_1").Return(batch.ScannedAt, true, nil),
		s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, "bt_a").Return(batch.ScannedAt, true, nil),
		s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, "bt_b").Return(batch.ScannedAt, true, nil),
	)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), batch.ID).Return(int64(0), nil)
	s.inventoryMutRepo.EXPECT().CountAllocatedReceiptsForBatch(gomock.Any(), undoAccountID, batch.ID).Return(int64(0), nil)

	s.deletedRecordRepo.EXPECT().CreateInAccount(gomock.Any(), gomock.Any(), batch.ID, undoAccountID, gomock.Any()).Return(nil)
	s.batchRepo.EXPECT().Delete(gomock.Any(), undoAccountID, batch.ID).Return(&domain.BaseBatch{ID: batch.ID}, nil)
	s.runActivity.EXPECT().NotifyBatchDeleted(gomock.Any(), gomock.Any(), undoAccountID, batch).Return(nil)

	for _, inputID := range inputs {
		s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, inputID).Return(scannedBatch(inputID), nil)
		s.batchRepo.EXPECT().ReopenIfNotFullyUsed(gomock.Any(), undoAccountID, gomock.Any(), gomock.Any(), undoStepID).Return(nil)
	}

	// No production run on a batch a scan created, so nothing to reopen.
	_, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().Nil(apiErr)
	s.Require().Len(s.outbox.undoEvents(), 1)
}

func (s *BatchUndoTestSuite) TestCarriesTheScrapSnapshotWhenTheRunIsBuildingForAnOrder() {
	batch := scannedBatch("bt_1")
	orderID := "ord_1"

	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, batch.ID).Return(batch, nil)
	s.expectHeld(batch)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), batch.ID).Return(int64(0), nil)
	s.inventoryMutRepo.EXPECT().CountAllocatedReceiptsForBatch(gomock.Any(), undoAccountID, batch.ID).Return(int64(0), nil)
	s.batchRepo.EXPECT().FindInputBatchIDs(gomock.Any(), batch.ID).Return(nil, nil)
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), batch.ID).Return(&domain.LineageShortfall{
		ProductionRunID: undoRunID,
		Seconds:         decimal.NewFromInt(3),
		Waste:           decimal.NewFromInt(2),
	}, nil)
	s.orderRepo.EXPECT().FindIDByProductionRun(gomock.Any(), undoAccountID, undoRunID).Return(&orderID, nil)
	s.stepRepo.EXPECT().Find(gomock.Any(), undoAccountID, undoStepID).Return(&domain.ProductionStepDetail{
		ID: undoStepID,
		Production: domain.StepProduction{
			ProducedItem: domain.LightItem{ID: "it_product"},
			Quantity:     domain.BatchQuantity{Measure: decimal.NewFromInt(1), Unit: domain.LightUnit{ID: "each"}},
		},
	}, nil)
	s.deletedRecordRepo.EXPECT().CreateInAccount(gomock.Any(), gomock.Any(), batch.ID, undoAccountID, gomock.Any()).Return(nil)
	s.batchRepo.EXPECT().Delete(gomock.Any(), undoAccountID, batch.ID).Return(&domain.BaseBatch{ID: batch.ID}, nil)
	s.runActivity.EXPECT().NotifyBatchDeleted(gomock.Any(), gomock.Any(), undoAccountID, batch).Return(nil)

	_, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().Nil(apiErr)
	events := s.outbox.undoEvents()
	s.Require().Len(events, 1)
	s.Equal(orderID, events[0].OrderID)
	s.Equal("it_product", events[0].ProducedItemID)
	s.Equal("5", events[0].ShortfallMeasure)
	s.Equal("each", events[0].ShortfallUnitID)
}

func (s *BatchUndoTestSuite) TestBulkDeleteUndoesAChainFromItsDownstreamEnd() {
	parent := scannedBatch("bt_parent")
	child := scannedBatch("bt_child")

	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, parent.ID).Return(parent, nil).AnyTimes()
	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, child.ID).Return(child, nil).AnyTimes()

	// The child is fed by the parent, so it has to be undone first.
	s.batchRepo.EXPECT().FindInputBatchIDs(gomock.Any(), parent.ID).Return(nil, nil).AnyTimes()
	s.batchRepo.EXPECT().FindInputBatchIDs(gomock.Any(), child.ID).Return([]string{parent.ID}, nil).AnyTimes()

	var undone []string
	s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, gomock.Any()).Return(parent.ScannedAt, true, nil).Times(3)
	s.batchRepo.EXPECT().CountDownstreamBatches(gomock.Any(), gomock.Any()).Return(int64(0), nil).Times(2)
	s.inventoryMutRepo.EXPECT().CountAllocatedReceiptsForBatch(gomock.Any(), undoAccountID, gomock.Any()).Return(int64(0), nil).Times(2)
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), gomock.Any()).
		Return(&domain.LineageShortfall{Seconds: decimal.Zero, Waste: decimal.Zero}, nil).Times(2)
	s.stepRepo.EXPECT().FindProducedUnit(gomock.Any(), undoAccountID, undoStepID).Return(&domain.LightUnit{ID: "each"}, nil).AnyTimes()
	s.batchRepo.EXPECT().ReopenIfNotFullyUsed(gomock.Any(), undoAccountID, gomock.Any(), gomock.Any(), undoStepID).Return(nil).AnyTimes()
	s.deletedRecordRepo.EXPECT().CreateInAccount(gomock.Any(), gomock.Any(), gomock.Any(), undoAccountID, gomock.Any()).Return(nil).Times(2)
	s.batchRepo.EXPECT().Delete(gomock.Any(), undoAccountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, batchID string) (*domain.BaseBatch, *apierror.APIError) {
			undone = append(undone, batchID)
			return &domain.BaseBatch{ID: batchID}, nil
		}).Times(2)
	s.runActivity.EXPECT().NotifyBatchDeleted(gomock.Any(), gomock.Any(), undoAccountID, child).Return(nil)
	s.runActivity.EXPECT().NotifyBatchDeleted(gomock.Any(), gomock.Any(), undoAccountID, parent).Return(nil)

	// Listed parent-first on purpose: undoing the parent while the child still feeds on it would be refused.
	apiErr := s.svc.DeleteManyBatches(undoIdentityCtx(), []string{parent.ID, child.ID})

	s.Require().Nil(apiErr)
	s.Equal([]string{child.ID, parent.ID}, undone)
}

// Two undos of one batch at once: the one that waited for the other's row finds the batch gone and
// answers as deleting it again would.
func (s *BatchUndoTestSuite) TestAnUndoThatWaitedForAnotherFindsTheBatchDeleted() {
	batch := scannedBatch("bt_1")
	s.expectScanUndoPlanned(batch)
	s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, batch.ID).Return(nil, false, nil)
	s.batchRepo.EXPECT().Find(gomock.Any(), undoAccountID, batch.ID).Return(nil, apierror.NewResourceNotFoundError("Batch not found."))
	s.deletedRecordRepo.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeBatch, batch.ID, undoAccountID).Return(true, nil)

	_, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().NotNil(apiErr)
	s.Equal("This batch has already been deleted and can no longer be modified.", apiErr.PublicMessage)
	s.Empty(s.outbox.undoEvents())
}

// An init-station batch undone and scanned again while this undo waited is not the scan it was asked to undo.
func (s *BatchUndoTestSuite) TestRefusesABatchScannedAgainWhileTheUndoWaited() {
	batch := scannedBatch("bt_1", inRun())
	s.expectScanUndoPlanned(batch)
	rescannedAt := batch.ScannedAt.Add(time.Minute)
	s.batchRepo.EXPECT().LockScan(gomock.Any(), undoAccountID, batch.ID).Return(&rescannedAt, true, nil)

	_, apiErr := s.svc.DeleteBatch(undoIdentityCtx(), batch.ID)

	s.Require().NotNil(apiErr)
	s.Equal("This batch changed while it was being undone. Refresh and try again.", apiErr.PublicMessage)
	s.Empty(s.outbox.undoEvents())
}
