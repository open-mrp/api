package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestBurnRateTimeSpanDays(t *testing.T) {
	t.Parallel()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, 1.0, burnRateTimeSpanDays(base, base))
	assert.InDelta(t, 2.0, burnRateTimeSpanDays(base, base.Add(48*time.Hour)), 0.001)
	assert.InDelta(t, 0.5, burnRateTimeSpanDays(base, base.Add(12*time.Hour)), 0.001)
}

type BurnRateMedTestSuite struct {
	suite.Suite
	med         domain.BurnRateMed
	itemRepo    *repositorymock.MockItemRepo
	rateRepo    *repositorymock.MockRateRepo
	unitConv    *repositorymock.MockUnitConversionRepo
	outbox      *recordingOutboxRepo
	repoFactory *factorymock.MockRepoFactory
	ctrl        *gomock.Controller
}

// recordingOutboxRepo captures enqueued messages so gate tests can assert whether a recalc was queued.
type recordingOutboxRepo struct {
	messages []messaging.OutboxMessageInput
}

func (r *recordingOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	r.messages = append(r.messages, input)
	return int64(len(r.messages)), nil
}

const (
	testAccountID  = "ac_burn123"
	testItemID     = "item_burn123"
	testCategoryID = "cat_burn123"
	testBurnRateID = "rate_burn123"
	testBaseUnitID = "unit_base123"
)

func (suite *BurnRateMedTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.itemRepo = repositorymock.NewMockItemRepo(suite.ctrl)
	suite.rateRepo = repositorymock.NewMockRateRepo(suite.ctrl)
	suite.unitConv = repositorymock.NewMockUnitConversionRepo(suite.ctrl)
	suite.outbox = &recordingOutboxRepo{}
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewItemRepo().Return(suite.itemRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewRateRepo().Return(suite.rateRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUnitConversionRepo().Return(suite.unitConv).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(suite.outbox).AnyTimes()

	suite.med = NewBurnRateMed(&BurnRateMedConfig{Repos: suite.repoFactory})
}

func (suite *BurnRateMedTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestBurnRateMedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BurnRateMedTestSuite))
}

// expectItemAndBaseUnit sets up the item load and category base-unit resolution common to every path.
func (suite *BurnRateMedTestSuite) expectItemAndBaseUnit() {
	suite.itemRepo.EXPECT().
		Get(gomock.Any(), domain.GetItemParams{AccountID: testAccountID, ItemID: testItemID}).
		Return(&domain.Item{ID: testItemID, ItemCategoryID: testCategoryID, BurnRateID: testBurnRateID}, nil)
	suite.itemRepo.EXPECT().
		GetCategoryBaseUnitID(gomock.Any(), testCategoryID).
		Return(testBaseUnitID, "", nil)
}

// A freshness-only touch keeps value and units untouched: only the RateID is populated.
var freshnessTouchParams = domain.UpdateRateParams{RateID: testBurnRateID}

// An item with fewer than two consumption logs cannot yield a new rate, but it must still be marked
// fresh so the stale-item sweep stops re-selecting it. This is the regression that caused the runaway.
func (suite *BurnRateMedTestSuite) TestRecalculate_InsufficientHistory_MarksFresh() {
	suite.expectItemAndBaseUnit()
	suite.itemRepo.EXPECT().
		ListConsumptionChangeLogsForBurnRate(gomock.Any(), testAccountID, testItemID).
		Return([]domain.BurnRateConsumptionLog{{Value: "-5", UnitID: testBaseUnitID, CreatedAt: time.Now()}}, nil)

	suite.rateRepo.EXPECT().Update(gomock.Any(), freshnessTouchParams).Return(&domain.Rate{ID: testBurnRateID}, nil)

	apiErr := suite.med.RecalculateFromHistory(context.Background(), testAccountID, testItemID)
	assert.Nil(suite.T(), apiErr)
}

// Consumption logs that sum to zero produce no meaningful rate but must still bump freshness.
func (suite *BurnRateMedTestSuite) TestRecalculate_ZeroConsumption_MarksFresh() {
	suite.expectItemAndBaseUnit()
	now := time.Now()
	suite.itemRepo.EXPECT().
		ListConsumptionChangeLogsForBurnRate(gomock.Any(), testAccountID, testItemID).
		Return([]domain.BurnRateConsumptionLog{
			{Value: "0", UnitID: testBaseUnitID, CreatedAt: now},
			{Value: "0", UnitID: testBaseUnitID, CreatedAt: now.Add(24 * time.Hour)},
		}, nil)
	suite.unitConv.EXPECT().
		ConvertValue(gomock.Any(), gomock.Any(), testBaseUnitID, testBaseUnitID).
		DoAndReturn(func(_ context.Context, m decimal.Decimal, _, _ string) (decimal.Decimal, *apierror.APIError) {
			return m, nil
		}).Times(2)

	suite.rateRepo.EXPECT().Update(gomock.Any(), freshnessTouchParams).Return(&domain.Rate{ID: testBurnRateID}, nil)

	apiErr := suite.med.RecalculateFromHistory(context.Background(), testAccountID, testItemID)
	assert.Nil(suite.T(), apiErr)
}

// With real consumption over a known span, the computed per-day rate is written to the burn rate,
// which also advances updated_at via UpdateRateByID's NOW(3).
func (suite *BurnRateMedTestSuite) TestRecalculate_ComputesAndWritesRate() {
	suite.expectItemAndBaseUnit()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	suite.itemRepo.EXPECT().
		ListConsumptionChangeLogsForBurnRate(gomock.Any(), testAccountID, testItemID).
		Return([]domain.BurnRateConsumptionLog{
			{Value: "-10", UnitID: testBaseUnitID, CreatedAt: start},
			{Value: "-10", UnitID: testBaseUnitID, CreatedAt: start.Add(48 * time.Hour)},
		}, nil)
	suite.unitConv.EXPECT().
		ConvertValue(gomock.Any(), gomock.Any(), testBaseUnitID, testBaseUnitID).
		DoAndReturn(func(_ context.Context, m decimal.Decimal, _, _ string) (decimal.Decimal, *apierror.APIError) {
			return m, nil
		}).Times(2)

	// Total consumption 20 over a 2-day span => 10/day, in base units per day.
	suite.rateRepo.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(domain.UpdateRateParams{})).
		DoAndReturn(func(_ context.Context, params domain.UpdateRateParams) (*domain.Rate, *apierror.APIError) {
			assert.Equal(suite.T(), testBurnRateID, params.RateID)
			suite.Require().NotNil(params.Value)
			assert.Equal(suite.T(), "10", *params.Value)
			suite.Require().NotNil(params.NumeratorUnitID)
			assert.Equal(suite.T(), testBaseUnitID, *params.NumeratorUnitID)
			suite.Require().NotNil(params.DenominatorUnitID)
			assert.Equal(suite.T(), burnRateDenominatorUnitID, *params.DenominatorUnitID)
			return &domain.Rate{ID: testBurnRateID}, nil
		})

	apiErr := suite.med.RecalculateFromHistory(context.Background(), testAccountID, testItemID)
	assert.Nil(suite.T(), apiErr)
}

// MaybeRecalculateAfterConsumption must enqueue a recalc only for genuine consumption: a negative
// delta booked as 'scan' (materials/parts) or 'system_action' (product fulfillment). 'user_correction'
// is a manual re-baseline, not demand, and must not trigger a recalc; nor may non-negative deltas.
// This gate mirrors ListConsumptionChangeLogsForBurnRate's SQL filter — keep the two in lockstep.
func (suite *BurnRateMedTestSuite) TestMaybeRecalculateAfterConsumption_ActionTypeGate() {
	cases := []struct {
		name        string
		actionType  string
		delta       decimal.Decimal
		wantEnqueue bool
	}{
		{"scan consumption enqueues", "scan", decimal.NewFromInt(-5), true},
		{"system_action consumption enqueues", "system_action", decimal.NewFromInt(-10), true},
		{"user_correction is excluded", "user_correction", decimal.NewFromInt(-5000), false},
		{"unknown action type is excluded", "create_record", decimal.NewFromInt(-5), false},
		{"positive delta never enqueues", "scan", decimal.NewFromInt(5), false},
		{"zero delta never enqueues", "system_action", decimal.Zero, false},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.outbox.messages = nil
			MaybeRecalculateAfterConsumption(context.Background(), suite.repoFactory, testAccountID, testItemID, tc.delta, tc.actionType)
			if tc.wantEnqueue {
				suite.Require().Len(suite.outbox.messages, 1)
				assert.Equal(suite.T(), string(contracts.CoreCmdRecalcItemBurnRate), suite.outbox.messages[0].MessageType)
				assert.Equal(suite.T(), string(contracts.CoreCmdRecalcItemBurnRate), suite.outbox.messages[0].RoutingKey)
			} else {
				assert.Empty(suite.T(), suite.outbox.messages)
			}
		})
	}
}
