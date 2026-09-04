package event

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	undoTestAccountID = "acct_test"
	undoTestBatchID   = "bt_1"
	undoTestProductID = "it_product"
	undoTestMaterial  = "it_material"
	undoTestOrderID   = "ord_1"
)

// stubTxManager runs the callback inline against the mocked factory.
//
// The mocks issue no SQL, so there is no transaction to open and nothing this can verify about
// atomicity — that is what the real-MySQL concurrency tests are for. What it does check is that the
// consumer reaches its repositories through the factory the transaction hands it rather than through
// the one on the struct, which is the whole substance of giving this consumer a transaction.
type stubTxManager struct {
	factory domain.RepoFactory
	// calls counts the transactions the consumer opened. The reversal and the allocation request it
	// justifies belong to the same one; see TestAllocationIsRequestedInsideTheReversalTransaction.
	calls int
}

func (m *stubTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	m.calls++
	return fn(ctx, m.factory)
}

func (m *stubTxManager) WithTxSavepoint(ctx context.Context, fn func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.factory, passthroughSavepoint{})
}

type passthroughSavepoint struct{}

func (passthroughSavepoint) Run(ctx context.Context, fn func(context.Context) *apierror.APIError) *apierror.APIError {
	return fn(ctx)
}

type UndoBatchScanConsumerTestSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	consumer         *UndoBatchScanConsumer
	inventoryMutRepo *repositorymock.MockInventoryMutationRepo
	reservationRepo  *repositorymock.MockInventoryReservationRepo
	materialRepo     *repositorymock.MockMaterialDemandRepo
	inventoryQuery   *repositorymock.MockInventoryQueryRepo
	txManager        *stubTxManager
	requested        []string
}

func (s *UndoBatchScanConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.inventoryMutRepo = repositorymock.NewMockInventoryMutationRepo(s.ctrl)
	s.reservationRepo = repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	s.materialRepo = repositorymock.NewMockMaterialDemandRepo(s.ctrl)
	s.inventoryQuery = repositorymock.NewMockInventoryQueryRepo(s.ctrl)

	// The reversal writes an audit trail; these keep it from failing the test without making it the subject of one.
	itemRepo := repositorymock.NewMockItemRepo(s.ctrl)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("item")).AnyTimes()
	s.inventoryQuery.EXPECT().FetchPhysicalInventory(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(decimal.Zero, nil).AnyTimes()
	s.inventoryMutRepo.EXPECT().CreateInventoryLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.inventoryMutRepo.EXPECT().CreateInventoryChangeLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	repoFactory := factorymock.NewMockRepoFactory(s.ctrl)
	repoFactory.EXPECT().NewInventoryMutationRepo().Return(s.inventoryMutRepo).AnyTimes()
	// The item set is resolved before the transaction, and its roots are the transaction's first
	// statements.
	s.inventoryMutRepo.EXPECT().ListItemIDsForBatchReversal(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]string{undoTestProductID, undoTestMaterial}, nil).AnyTimes()
	s.inventoryMutRepo.EXPECT().LockItemForLedger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	repoFactory.EXPECT().NewInventoryReservationRepo().Return(s.reservationRepo).AnyTimes()
	repoFactory.EXPECT().NewMaterialDemandRepo().Return(s.materialRepo).AnyTimes()
	repoFactory.EXPECT().NewInventoryQueryRepo().Return(s.inventoryQuery).AnyTimes()
	repoFactory.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	s.requested = nil
	s.txManager = &stubTxManager{factory: repoFactory}
	repoFactory.EXPECT().NewOutboxRepo().Return(recordingOutboxRepo{
		onAllocateOpenIssues: func(itemID string) { s.requested = append(s.requested, itemID) },
	}).AnyTimes()

	s.consumer = &UndoBatchScanConsumer{
		repos:     repoFactory,
		txManager: s.txManager,
		tracer:    tracing.GetTracer("test.undo_batch_scan_consumer"),
	}
}

func (s *UndoBatchScanConsumerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestUndoBatchScanConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UndoBatchScanConsumerTestSuite))
}

// The reversal frees receipts, so every item it touched needs its open demand looked at again.
//
// It asks rather than does: covering the demand inline walked every open issue of every reversed item
// inside this consumer, in the opposite lock order from the allocate consumer doing the same work.
// The request is sorted, because two of these taking the same items in different orders is a deadlock
// nobody could explain from the logs.
func (s *UndoBatchScanConsumerTestSuite) TestRequestsAllocationForEveryItemTheReversalTouched() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), domain.ReverseInventoryForBatchParams{
		AccountID: undoTestAccountID,
		BatchID:   undoTestBatchID,
	}).Return([]domain.InventoryReversalDelta{
		{ItemID: undoTestProductID, Measure: decimal.NewFromInt(-10), UnitID: "each"},
		{ItemID: undoTestMaterial, Measure: decimal.NewFromInt(40), UnitID: "each"},
	}, nil)

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().NoError(err)
	s.Equal(ledgerlock.SortedUnique([]string{undoTestProductID, undoTestMaterial}), s.requested)
}

func (s *UndoBatchScanConsumerTestSuite) TestRefusalIsReportedRatherThanSwallowed() {
	// The delete checked that nothing had drawn on the output, but that was before this message was
	// picked up. Returning the error parks it in the dead-letter queue instead of half-reversing.
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewValidationError("Inventory produced by this batch has already been used and cannot be reversed."))

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().Error(err)
	apiErr, ok := err.(*apierror.APIError)
	s.Require().True(ok)
	s.Contains(apiErr.PublicMessage, "already been used")
}

func (s *UndoBatchScanConsumerTestSuite) TestRestoresTheScrapReservationsTheScanReleased() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	s.materialRepo.EXPECT().GetMaterialDemand(gomock.Any(), undoTestAccountID, undoTestProductID, decimal.NewFromInt(5), "each").
		Return([]domain.MaterialDemandItem{
			{ItemID: undoTestMaterial, Measure: decimal.NewFromInt(10), UnitID: "each"},
		}, nil)

	var reserved []domain.CreateMaterialReservationParams
	s.reservationRepo.EXPECT().CreateMaterialReservation(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ *ledgerlock.Scope, params domain.CreateMaterialReservationParams) *apierror.APIError {
			reserved = append(reserved, params)
			return nil
		}).Times(2)

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID:          undoTestBatchID,
		OrderID:          undoTestOrderID,
		ProducedItemID:   undoTestProductID,
		ShortfallMeasure: "5",
		ShortfallUnitID:  "each",
	})

	s.Require().NoError(err)
	s.Require().Len(reserved, 2)

	// The units that will never be produced go back on hold for the order...
	s.Equal(undoTestProductID, reserved[0].ItemID)
	s.Equal("5", reserved[0].Measure.String())
	s.Equal(undoTestOrderID, reserved[0].OrderID)

	// ...and so do the materials that would have gone into them.
	s.Equal(undoTestMaterial, reserved[1].ItemID)
	s.Equal("10", reserved[1].Measure.String())
}

func (s *UndoBatchScanConsumerTestSuite) TestLeavesReservationsAloneWhenTheScanReleasedNone() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	// No CreateMaterialReservation or GetMaterialDemand expectations: calling either fails the test.
	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().NoError(err)
}

func (s *UndoBatchScanConsumerTestSuite) TestIgnoresAShortfallItCannotParse() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID:          undoTestBatchID,
		OrderID:          undoTestOrderID,
		ProducedItemID:   undoTestProductID,
		ShortfallMeasure: "not-a-number",
		ShortfallUnitID:  "each",
	})

	s.Require().NoError(err)
}

// The allocation request rides with the reversal rather than following it in a transaction of its own.
//
// The outbox row commits with the reversal, which is the guarantee wanted here: the request exists if
// and only if the stock was actually freed. Split into a second transaction after the inbox recovery
// point, a failure there could not be retried — the record is already 'processed', so the redelivery
// is skipped, MarkFailed is refused on a terminal record, and the failure monitor scans neither. The
// freed receipts would sit uncovered with nothing anywhere saying so.
func (s *UndoBatchScanConsumerTestSuite) TestAllocationIsRequestedInsideTheReversalTransaction() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]domain.InventoryReversalDelta{
			{ItemID: undoTestProductID, Measure: decimal.NewFromInt(-10), UnitID: "each"},
		}, nil)

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().NoError(err)
	s.Equal([]string{undoTestProductID}, s.requested, "the reversal asks for the item to be re-covered")
	s.Equal(1, s.txManager.calls,
		"one transaction: a second one after the recovery point cannot be retried if it fails")
}

// A reversal that never happened must not leave an allocation request behind asking to cover demand
// from stock that was never freed.
func (s *UndoBatchScanConsumerTestSuite) TestNoAllocationIsRequestedWhenTheReversalFails() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewValidationError("Inventory produced by this batch has already been used and cannot be reversed."))

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().Error(err)
	s.Empty(s.requested, "nothing was freed, so nothing is asked to be covered")
}
