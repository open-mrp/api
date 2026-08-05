package event

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

const (
	undoTestAccountID = "acct_test"
	undoTestBatchID   = "bt_1"
	undoTestProductID = "it_product"
	undoTestMaterial  = "it_material"
	undoTestOrderID   = "ord_1"
)

type UndoBatchScanConsumerTestSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	consumer         *UndoBatchScanConsumer
	inventoryMutRepo *repositorymock.MockInventoryMutationRepo
	reservationRepo  *repositorymock.MockInventoryReservationRepo
	materialRepo     *repositorymock.MockMaterialDemandRepo
	inventoryQuery   *repositorymock.MockInventoryQueryRepo
}

func (s *UndoBatchScanConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.inventoryMutRepo = repositorymock.NewMockInventoryMutationRepo(s.ctrl)
	s.reservationRepo = repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	s.materialRepo = repositorymock.NewMockMaterialDemandRepo(s.ctrl)
	s.inventoryQuery = repositorymock.NewMockInventoryQueryRepo(s.ctrl)

	// The audit trail is written best-effort behind the reversal; these keep it from failing the
	// test without making it the subject of one.
	itemRepo := repositorymock.NewMockItemRepo(s.ctrl)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("item")).AnyTimes()
	s.inventoryQuery.EXPECT().FetchPhysicalInventory(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(float64(0), nil).AnyTimes()
	s.inventoryMutRepo.EXPECT().CreateInventoryLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.inventoryMutRepo.EXPECT().CreateInventoryChangeLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	repoFactory := factorymock.NewMockRepoFactory(s.ctrl)
	repoFactory.EXPECT().NewInventoryMutationRepo().Return(s.inventoryMutRepo).AnyTimes()
	repoFactory.EXPECT().NewInventoryReservationRepo().Return(s.reservationRepo).AnyTimes()
	repoFactory.EXPECT().NewMaterialDemandRepo().Return(s.materialRepo).AnyTimes()
	repoFactory.EXPECT().NewInventoryQueryRepo().Return(s.inventoryQuery).AnyTimes()
	repoFactory.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()

	s.consumer = &UndoBatchScanConsumer{
		repos:  repoFactory,
		tracer: tracing.GetTracer("test.undo_batch_scan_consumer"),
	}
}

func (s *UndoBatchScanConsumerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestUndoBatchScanConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UndoBatchScanConsumerTestSuite))
}

func (s *UndoBatchScanConsumerTestSuite) TestReAllocatesEveryItemTheReversalTouched() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), domain.ReverseInventoryForBatchParams{
		AccountID: undoTestAccountID,
		BatchID:   undoTestBatchID,
	}).Return([]domain.InventoryReversalDelta{
		{ItemID: undoTestProductID, Measure: decimal.NewFromInt(-10), UnitID: "each"},
		{ItemID: undoTestMaterial, Measure: decimal.NewFromInt(40), UnitID: "each"},
	}, nil)

	allocated := map[string]bool{}
	s.reservationRepo.EXPECT().AllocateOpenIssuesForItem(gomock.Any(), undoTestAccountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, itemID string) *apierror.APIError {
			allocated[itemID] = true
			return nil
		}).Times(2)

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().NoError(err)
	s.Equal(map[string]bool{undoTestProductID: true, undoTestMaterial: true}, allocated)
}

func (s *UndoBatchScanConsumerTestSuite) TestRefusalIsReportedRatherThanSwallowed() {
	// The delete checked that nothing had drawn on the output, but that was before this message was
	// picked up. Returning the error parks it in the dead-letter queue instead of half-reversing.
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any()).
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
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any()).Return(nil, nil)

	s.materialRepo.EXPECT().GetMaterialDemand(gomock.Any(), undoTestAccountID, undoTestProductID, decimal.NewFromInt(5), "each").
		Return([]domain.MaterialDemandItem{
			{ItemID: undoTestMaterial, Measure: decimal.NewFromInt(10), UnitID: "each"},
		}, nil)

	var reserved []domain.CreateMaterialReservationParams
	s.reservationRepo.EXPECT().CreateMaterialReservation(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.CreateMaterialReservationParams) *apierror.APIError {
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
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any()).Return(nil, nil)

	// No CreateMaterialReservation or GetMaterialDemand expectations: calling either fails the test.
	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID: undoTestBatchID,
	})

	s.Require().NoError(err)
}

func (s *UndoBatchScanConsumerTestSuite) TestIgnoresAShortfallItCannotParse() {
	s.inventoryMutRepo.EXPECT().ReverseInventoryForBatch(gomock.Any(), gomock.Any()).Return(nil, nil)

	err := s.consumer.undoBatchScan(context.Background(), undoTestAccountID, domain.UndoBatchScanEvent{
		BatchID:          undoTestBatchID,
		OrderID:          undoTestOrderID,
		ProducedItemID:   undoTestProductID,
		ShortfallMeasure: "not-a-number",
		ShortfallUnitID:  "each",
	})

	s.Require().NoError(err)
}
