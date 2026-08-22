package event

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	scanAccountID  = "acct_test"
	scanBatchID    = "bt_1"
	scanStepID     = "ps_knit_631"
	scanStationID  = "sgsn_knit"
	scanProducedID = "it_greige_631"
	scanYarnID     = "it_yarn"
	scanDyeID      = "it_dye"
	scanOrderID    = "ord_1"
	scanRunID      = "prun_1"

	unitPair  = "un_pair"
	unitDozen = "un_dozen"
	unitPound = "un_pound"
)

type BatchScannedConsumerTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	consumer        *BatchScannedConsumer
	stepQueryRepo   *repositorymock.MockProductionStepQueryRepo
	unitConvRepo    *repositorymock.MockUnitConversionRepo
	inventoryMuts   *repositorymock.MockInventoryMutationRepo
	reservationRepo *repositorymock.MockInventoryReservationRepo
	batchRepo       *repositorymock.MockBatchRepo
	orderQueryRepo  *repositorymock.MockOrderQueryRepo
	materialRepo    *repositorymock.MockMaterialDemandRepo

	// inventoryMoves records every movement the scan produced, keyed by item, so a test can assert on
	// the arithmetic without caring what order the consumer applied it in.
	inventoryMoves map[string]decimal.Decimal
	moveUnits      map[string]string
	// allocatedItems records, in order, the items offered to outstanding open issues.
	allocatedItems []string
}

func (s *BatchScannedConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.inventoryMoves = map[string]decimal.Decimal{}
	s.moveUnits = map[string]string{}
	s.allocatedItems = nil

	s.stepQueryRepo = repositorymock.NewMockProductionStepQueryRepo(s.ctrl)
	s.unitConvRepo = repositorymock.NewMockUnitConversionRepo(s.ctrl)
	s.inventoryMuts = repositorymock.NewMockInventoryMutationRepo(s.ctrl)
	s.reservationRepo = repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	s.batchRepo = repositorymock.NewMockBatchRepo(s.ctrl)
	s.orderQueryRepo = repositorymock.NewMockOrderQueryRepo(s.ctrl)
	s.materialRepo = repositorymock.NewMockMaterialDemandRepo(s.ctrl)

	s.inventoryMuts.EXPECT().UpdateInventory(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.InventoryUpdateParams) *apierror.APIError {
			prev, ok := s.inventoryMoves[p.ItemID]
			if !ok {
				prev = decimal.Zero
			}
			s.inventoryMoves[p.ItemID] = prev.Add(p.Measure)
			s.moveUnits[p.ItemID] = p.UnitID
			return nil
		}).AnyTimes()

	// The audit trail runs best-effort behind each movement; these keep it quiet rather than making
	// it the subject of these tests.
	itemRepo := repositorymock.NewMockItemRepo(s.ctrl)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("item")).AnyTimes()
	inventoryQuery := repositorymock.NewMockInventoryQueryRepo(s.ctrl)
	inventoryQuery.EXPECT().FetchPhysicalInventory(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(decimal.Zero, nil).AnyTimes()
	s.inventoryMuts.EXPECT().CreateInventoryLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.inventoryMuts.EXPECT().CreateInventoryChangeLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	repoFactory := factorymock.NewMockRepoFactory(s.ctrl)
	repoFactory.EXPECT().NewProductionStepQueryRepo().Return(s.stepQueryRepo).AnyTimes()
	repoFactory.EXPECT().NewUnitConversionRepo().Return(s.unitConvRepo).AnyTimes()
	repoFactory.EXPECT().NewInventoryMutationRepo().Return(s.inventoryMuts).AnyTimes()
	repoFactory.EXPECT().NewInventoryReservationRepo().Return(s.reservationRepo).AnyTimes()
	repoFactory.EXPECT().NewBatchRepo().Return(s.batchRepo).AnyTimes()
	repoFactory.EXPECT().NewOrderQueryRepo().Return(s.orderQueryRepo).AnyTimes()
	repoFactory.EXPECT().NewMaterialDemandRepo().Return(s.materialRepo).AnyTimes()
	repoFactory.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repoFactory.EXPECT().NewInventoryQueryRepo().Return(inventoryQuery).AnyTimes()

	// Allocation of outstanding shortages closes every scan; the tests that are about it assert on
	// s.allocatedItems, the rest just need it not to fail.
	s.reservationRepo.EXPECT().AllocateOpenIssuesForItem(gomock.Any(), scanAccountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, itemID string) *apierror.APIError {
			s.allocatedItems = append(s.allocatedItems, itemID)
			return nil
		}).AnyTimes()

	s.consumer = &BatchScannedConsumer{
		repos:  repoFactory,
		tracer: tracing.GetTracer("test.batch_scanned_consumer"),
	}
}

func (s *BatchScannedConsumerTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestBatchScannedConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BatchScannedConsumerTestSuite))
}

// ─── helpers ───────────────────────────────────────────────────────────────

func qty(measure string, unitID string) domain.BatchQuantity {
	return domain.BatchQuantity{
		Measure: decimal.RequireFromString(measure),
		Unit:    domain.LightUnit{ID: unitID},
	}
}

// step builds a production step producing `producedMeasure` of the greige item per execution.
func step(producedMeasure string, producedUnit string, consumptions ...domain.StepConsumption) *domain.ProductionStepDetail {
	return &domain.ProductionStepDetail{
		ID: scanStepID,
		Production: domain.StepProduction{
			ProducedItem: domain.LightItem{ID: scanProducedID},
			Quantity:     qty(producedMeasure, producedUnit),
		},
		Consumptions: consumptions,
	}
}

func consumes(itemID, measure, unitID string) domain.StepConsumption {
	return domain.StepConsumption{
		ConsumedItem:  domain.LightItem{ID: itemID},
		Quantity:      qty(measure, unitID),
		WasteQuantity: qty("0", unitID),
	}
}

func scanEvent(measure, unitID string) domain.BatchScannedEvent {
	return domain.BatchScannedEvent{
		AccountID:         scanAccountID,
		BatchID:           scanBatchID,
		ProductionStepID:  scanStepID,
		ScanningStationID: scanStationID,
		ItemID:            scanProducedID,
		Measure:           measure,
		UnitID:            unitID,
		ScannedAt:         time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
}

// expectConversion stands in for the unit repo, returning `to` whenever the scanned measure is
// converted into the step's unit.
func (s *BatchScannedConsumerTestSuite) expectConversion(from, to string) {
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), decimal.RequireFromString(from), gomock.Any(), gomock.Any()).
		Return(decimal.RequireFromString(to), nil).AnyTimes()
}

// expectNoProductionRun makes the batch look like it is being built to stock.
func (s *BatchScannedConsumerTestSuite) expectBuiltToStock() {
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), scanBatchID).
		Return(&domain.LineageShortfall{}, nil).AnyTimes()
}

func (s *BatchScannedConsumerTestSuite) moved(itemID string) decimal.Decimal {
	v, ok := s.inventoryMoves[itemID]
	if !ok {
		return decimal.Zero
	}
	return v
}

// ─── production arithmetic ─────────────────────────────────────────────────

// A scan in the step's own unit produces exactly what was scanned.
func (s *BatchScannedConsumerTestSuite) TestProducesScannedQuantityWhenUnitsMatch() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.expectConversion("60", "60")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("60", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("60").Equal(s.moved(scanProducedID)),
		"expected 60 produced, got %s", s.moved(scanProducedID))
	s.Equal(unitPair, s.moveUnits[scanProducedID])
}

// The station scans dozens while the step is written in pairs. The conversion happens once and
// everything downstream is in the step's unit — the receipt must be in pairs, not dozens.
func (s *BatchScannedConsumerTestSuite) TestConvertsScannedUnitIntoStepUnit() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	// 5 dozen scanned = 60 pairs.
	s.unitConvRepo.EXPECT().
		ConvertValue(gomock.Any(), decimal.RequireFromString("5"), unitDozen, unitPair).
		Return(decimal.RequireFromString("60"), nil)
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("5", unitDozen))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("60").Equal(s.moved(scanProducedID)),
		"expected 60 pairs produced, got %s", s.moved(scanProducedID))
	s.Equal(unitPair, s.moveUnits[scanProducedID], "the receipt is recorded in the step's unit")
}

// A scan of less than one execution of the step still produces its fraction rather than rounding to
// a whole execution.
func (s *BatchScannedConsumerTestSuite) TestProducesFractionalExecution() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.expectConversion("6", "6")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("6", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("6").Equal(s.moved(scanProducedID)))
}

// A non-terminating multiplier must not drift: 10 scanned against a step of 3 is 10 produced, not
// 9.999… from 3 × (10/3) evaluated in floating point.
func (s *BatchScannedConsumerTestSuite) TestNonTerminatingMultiplierDoesNotDrift() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("3", unitPair), nil)
	s.expectConversion("10", "10")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("10", unitPair))

	s.Require().Nil(err)
	produced := s.moved(scanProducedID)
	// Whatever rounding the decimal division applies, the result must stay within a hair of 10.
	s.True(produced.Sub(decimal.RequireFromString("10")).Abs().LessThan(decimal.RequireFromString("0.0000000001")),
		"expected ~10 produced, got %s", produced)
}

// ─── consumption arithmetic ────────────────────────────────────────────────

// Consumption scales with the multiplier, not with the raw scanned number.
func (s *BatchScannedConsumerTestSuite) TestConsumptionScalesWithMultiplier() {
	// Step makes 12 pr and eats 4 lbs of yarn per execution. Scanning 60 pr is 5 executions → 20 lbs.
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "4", unitPound)), nil)
	s.expectConversion("60", "60")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("60", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("-20").Equal(s.moved(scanYarnID)),
		"expected 20 lbs consumed, got %s", s.moved(scanYarnID))
	s.Equal(unitPound, s.moveUnits[scanYarnID], "consumption stays in the consumption's own unit")
}

// Consumption waste is material the step burns without it reaching the product, and is drawn down
// alongside what the product takes.
func (s *BatchScannedConsumerTestSuite) TestConsumptionIncludesItsWaste() {
	c := consumes(scanYarnID, "4", unitPound)
	c.WasteQuantity = qty("1", unitPound)
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, c), nil)
	s.expectConversion("12", "12")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("-5").Equal(s.moved(scanYarnID)),
		"expected 4 + 1 waste consumed, got %s", s.moved(scanYarnID))
}

// Every consumption on the step is applied, each in its own unit.
func (s *BatchScannedConsumerTestSuite) TestAppliesEveryConsumption() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair,
			consumes(scanYarnID, "4", unitPound),
			consumes(scanDyeID, "2", unitPair),
		), nil)
	s.expectConversion("12", "12")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("-4").Equal(s.moved(scanYarnID)))
	s.True(decimal.RequireFromString("-2").Equal(s.moved(scanDyeID)))
	s.Equal(unitPound, s.moveUnits[scanYarnID])
	s.Equal(unitPair, s.moveUnits[scanDyeID])
}

// A consumption of zero writes nothing rather than a no-op ledger row.
func (s *BatchScannedConsumerTestSuite) TestZeroConsumptionIsSkipped() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "0", unitPound)), nil)
	s.expectConversion("12", "12")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	_, touched := s.inventoryMoves[scanYarnID]
	s.False(touched, "a zero consumption should not write a ledger row")
}

// ─── order-backed consumption ──────────────────────────────────────────────

// With an order behind the batch, material comes out of its reservation first.
func (s *BatchScannedConsumerTestSuite) TestConsumptionDrawsFromReservationFirst() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "4", unitPound)), nil)
	s.expectConversion("12", "12")
	s.expectOrderBackedBatch()

	// The reservation covered the whole 4 lbs.
	s.reservationRepo.EXPECT().AllocateReservationsForConsumption(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.ConsumptionAllocationParams) (*domain.ConsumptionAllocationResult, *apierror.APIError) {
			s.Equal(scanOrderID, p.OrderID)
			s.Equal(scanBatchID, p.ProducedBatchID, "the batch tag is what makes the consumption reversible")
			s.True(decimal.RequireFromString("4").Equal(p.Measure))
			return &domain.ConsumptionAllocationResult{RemainingMeasure: decimal.Zero, RemainingUnitID: unitPound}, nil
		})

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	_, touched := s.inventoryMoves[scanYarnID]
	s.False(touched, "a fully reserved consumption should not also come off open stock")
}

// Whatever the reservation could not cover is taken from open stock.
func (s *BatchScannedConsumerTestSuite) TestShortfallOnReservationComesOffOpenStock() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "4", unitPound)), nil)
	s.expectConversion("12", "12")
	s.expectOrderBackedBatch()

	// Only 1.5 of the 4 lbs was reserved.
	s.reservationRepo.EXPECT().AllocateReservationsForConsumption(gomock.Any(), gomock.Any()).
		Return(&domain.ConsumptionAllocationResult{
			RemainingMeasure: decimal.RequireFromString("2.5"),
			RemainingUnitID:  unitPound,
		}, nil)

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("-2.5").Equal(s.moved(scanYarnID)),
		"expected the 2.5 lb shortfall off open stock, got %s", s.moved(scanYarnID))
}

// expectOrderBackedBatch makes the batch look like it is being built against an order, with no scrap.
func (s *BatchScannedConsumerTestSuite) expectOrderBackedBatch() {
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), scanBatchID).
		Return(&domain.LineageShortfall{ProductionRunID: scanRunID}, nil)
	s.orderQueryRepo.EXPECT().FindIDByProductionRun(gomock.Any(), scanAccountID, scanRunID).
		Return(ptr(scanOrderID), nil)
}

// ─── seconds and waste ─────────────────────────────────────────────────────

// Seconds and waste release the reservation covering units the batch will never deliver, and the
// upstream material reserved to make them.
func (s *BatchScannedConsumerTestSuite) TestSecondsAndWasteReleaseReservations() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.expectConversion("60", "60")
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), scanBatchID).
		Return(&domain.LineageShortfall{
			ProductionRunID: scanRunID,
			Seconds:         decimal.RequireFromString("3"),
			Waste:           decimal.RequireFromString("2"),
		}, nil)
	s.orderQueryRepo.EXPECT().FindIDByProductionRun(gomock.Any(), scanAccountID, scanRunID).
		Return(ptr(scanOrderID), nil)

	s.reservationRepo.EXPECT().ReduceReservedForOrderItem(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.OrderReservationReductionParams) *apierror.APIError {
			s.True(decimal.RequireFromString("5").Equal(p.Measure), "seconds + waste, got %s", p.Measure)
			s.Equal(scanProducedID, p.ItemID)
			return nil
		})

	demands := []domain.MaterialDemandItem{{ItemID: scanYarnID, Measure: decimal.RequireFromString("2"), UnitID: unitPound}}
	s.materialRepo.EXPECT().GetMaterialDemand(gomock.Any(), scanAccountID, scanProducedID, gomock.Any(), unitPair).
		Return(demands, nil)
	s.reservationRepo.EXPECT().
		ReduceReservedForOrderMaterials(gomock.Any(), scanOrderID, scanAccountID, demands).Return(nil)

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("60", unitPair))

	s.Require().Nil(err)
	// The produced receipt is the full scanned quantity; the shortfall moves reservations, not stock.
	s.True(decimal.RequireFromString("60").Equal(s.moved(scanProducedID)))
}

// A batch built to stock has no reservation to release, so the shortfall is not chased further.
func (s *BatchScannedConsumerTestSuite) TestNoShortfallReleaseWithoutProductionRun() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.expectConversion("12", "12")
	s.batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), scanBatchID).
		Return(&domain.LineageShortfall{Seconds: decimal.RequireFromString("3")}, nil)

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
}

// ─── refusals ──────────────────────────────────────────────────────────────

// A step that no longer produces what was scanned means the routing moved under the batch. Silently
// acking would drop the scan's inventory, so this must surface.
func (s *BatchScannedConsumerTestSuite) TestRefusesWhenStepNoLongerProducesScannedItem() {
	other := step("12", unitPair)
	other.Production.ProducedItem.ID = "it_something_else"
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(other, nil)

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().NotNil(err)
	s.Empty(s.inventoryMoves, "nothing should move when the step and the scan disagree")
}

// A zero production quantity would make the multiplier a division by zero.
func (s *BatchScannedConsumerTestSuite) TestRefusesZeroProductionQuantity() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("0", unitPair), nil)

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().NotNil(err)
	s.Empty(s.inventoryMoves)
}

func (s *BatchScannedConsumerTestSuite) TestRefusesUnparseableMeasure() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("not-a-number", unitPair))

	s.Require().NotNil(err)
	s.Empty(s.inventoryMoves)
}

// A failed conversion must stop the scan rather than fall back to the unconverted number, which
// would credit dozens as pairs.
func (s *BatchScannedConsumerTestSuite) TestRefusesWhenConversionFails() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(decimal.Zero, apierror.NewValidationError("no conversion between un_dozen and un_pair"))

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("5", unitDozen))

	s.Require().NotNil(err)
	s.Empty(s.inventoryMoves)
}

// ─── payload ───────────────────────────────────────────────────────────────

func TestBatchScannedEventOptionalMeasures(t *testing.T) {
	t.Parallel()

	evt := domain.BatchScannedEvent{}
	seconds, err := evt.SecondsDecimal()
	if err != nil || !seconds.IsZero() {
		t.Fatalf("absent seconds should read as zero, got %s (err %v)", seconds, err)
	}
	waste, err := evt.WasteDecimal()
	if err != nil || !waste.IsZero() {
		t.Fatalf("absent waste should read as zero, got %s (err %v)", waste, err)
	}

	evt = domain.BatchScannedEvent{SecondsMeasure: "2.5", WasteMeasure: "1"}
	seconds, err = evt.SecondsDecimal()
	if err != nil || !seconds.Equal(decimal.RequireFromString("2.5")) {
		t.Fatalf("expected 2.5 seconds, got %s (err %v)", seconds, err)
	}
	waste, err = evt.WasteDecimal()
	if err != nil || !waste.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("expected 1 waste, got %s (err %v)", waste, err)
	}

	evt = domain.BatchScannedEvent{SecondsMeasure: "oops"}
	if _, err := evt.SecondsDecimal(); err == nil {
		t.Fatal("an unparseable seconds measure should be an error, not silently zero")
	}
}

func ptr[T any](v T) *T { return &v }

// ─── scrap and consumption parity ──────────────────────────────────────────

// Material is burned by everything the step ran, not just the part that came out saleable. The
// legacy path expressed this as a second scan carrying the scrap; here it rides on the one event,
// and the totals have to agree.
func (s *BatchScannedConsumerTestSuite) TestConsumptionCoversScrapAsWellAsGoodOutput() {
	// 12 pr per execution, 4 lbs of yarn per execution. 60 pr good + 12 pr scrap = 6 executions of
	// yarn (24 lbs), but only 5 executions of finished goods (60 pr).
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "4", unitPound)), nil)
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), decimal.RequireFromString("60"), unitPair, unitPair).
		Return(decimal.RequireFromString("60"), nil)
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), decimal.RequireFromString("12"), unitPair, unitPair).
		Return(decimal.RequireFromString("12"), nil)
	s.expectBuiltToStock()

	evt := scanEvent("60", unitPair)
	evt.SecondsMeasure = "5"
	evt.WasteMeasure = "7"

	err := s.consumer.applyInventory(context.Background(), scanAccountID, evt)

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("60").Equal(s.moved(scanProducedID)),
		"only good output is produced, got %s", s.moved(scanProducedID))
	s.True(decimal.RequireFromString("-24").Equal(s.moved(scanYarnID)),
		"yarn should cover good output and scrap, got %s", s.moved(scanYarnID))
}

// With no scrap the consumption multiplier is the production one, and no second conversion is needed.
func (s *BatchScannedConsumerTestSuite) TestNoScrapMeansOneConversion() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "4", unitPound)), nil)
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), decimal.RequireFromString("60"), unitPair, unitPair).
		Return(decimal.RequireFromString("60"), nil).Times(1)
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("60", unitPair))

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("-20").Equal(s.moved(scanYarnID)))
}

// Scrap is converted into the step's unit like the scanned measure is, so a station scanning dozens
// against a step in pairs does not burn twelve times too little material on its scrap.
func (s *BatchScannedConsumerTestSuite) TestScrapIsConvertedIntoStepUnit() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair, consumes(scanYarnID, "4", unitPound)), nil)
	// 5 dozen good = 60 pr; 1 dozen scrap = 12 pr. 72 pr total = 6 executions = 24 lbs.
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), decimal.RequireFromString("5"), unitDozen, unitPair).
		Return(decimal.RequireFromString("60"), nil)
	s.unitConvRepo.EXPECT().ConvertValue(gomock.Any(), decimal.RequireFromString("1"), unitDozen, unitPair).
		Return(decimal.RequireFromString("12"), nil)
	s.expectBuiltToStock()

	evt := scanEvent("5", unitDozen)
	evt.SecondsMeasure = "1"

	err := s.consumer.applyInventory(context.Background(), scanAccountID, evt)

	s.Require().Nil(err)
	s.True(decimal.RequireFromString("60").Equal(s.moved(scanProducedID)))
	s.True(decimal.RequireFromString("-24").Equal(s.moved(scanYarnID)),
		"expected 24 lbs for 6 executions, got %s", s.moved(scanYarnID))
}

// An unparseable scrap measure must stop the scan rather than be treated as no scrap, which would
// silently under-consume material.
func (s *BatchScannedConsumerTestSuite) TestRefusesUnparseableScrap() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.expectConversion("12", "12")

	evt := scanEvent("12", unitPair)
	evt.WasteMeasure = "not-a-number"

	err := s.consumer.applyInventory(context.Background(), scanAccountID, evt)

	s.Require().NotNil(err)
}

// ─── allocating what the scan produced ─────────────────────────────────────

// An issue goes open when demand outran the shelf. The scan is the stock arriving, so the produced
// item and every material it touched are offered to whatever is still short.
func (s *BatchScannedConsumerTestSuite) TestOffersProducedAndConsumedItemsToOpenIssues() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair,
			consumes(scanYarnID, "4", unitPound),
			consumes(scanDyeID, "1", unitPair),
		), nil)
	s.expectConversion("12", "12")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	s.Equal([]string{scanProducedID, scanYarnID, scanDyeID}, s.allocatedItems,
		"produced item first, then each consumed item")
}

// An item consumed twice by the same step is offered once.
func (s *BatchScannedConsumerTestSuite) TestOffersEachItemOnce() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).
		Return(step("12", unitPair,
			consumes(scanYarnID, "4", unitPound),
			consumes(scanYarnID, "1", unitPound),
		), nil)
	s.expectConversion("12", "12")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	s.Equal([]string{scanProducedID, scanYarnID}, s.allocatedItems)
}

// A step that produces without consuming still offers its output.
func (s *BatchScannedConsumerTestSuite) TestOffersProducedItemWithNoConsumptions() {
	s.stepQueryRepo.EXPECT().Find(gomock.Any(), scanAccountID, scanStepID).Return(step("12", unitPair), nil)
	s.expectConversion("12", "12")
	s.expectBuiltToStock()

	err := s.consumer.applyInventory(context.Background(), scanAccountID, scanEvent("12", unitPair))

	s.Require().Nil(err)
	s.Equal([]string{scanProducedID}, s.allocatedItems)
}
