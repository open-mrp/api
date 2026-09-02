package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

// A carton of eight, an each, and an item stocked by the carton whose only production step is written
// in eaches — the shape every corrupted SKU in the incident had.
const (
	testCostAccountID   = "ac_cost"
	testCostItemID      = "itm_hotmitt"
	testCostStepID      = "ps_sew"
	testCostUnitGroupID = "ug_case_of_eight"
	testCostUnitEach    = "un_ea"
	testCostUnitCarton  = "un_ct8ea"
)

// eachesPerCarton is what the unit group says, and the factor every assertion below turns on.
var eachesPerCarton = decimal.NewFromInt(8)

// rolloutStubs is the set of reads one pass of the rollup makes, wired so a test can vary the two
// things the defect turned on: the unit the step is written in, and the units on either side of a
// consumption.
type rolloutStubs struct {
	stepUnitID      string
	stepQuantity    decimal.Decimal
	consumptions    []domain.CostFlowConsumption
	stepUnitInGroup bool
}

// costRollupHarness wires a service against mock repositories and hands back the unit costs the
// rollup persisted, which is the thing the incident was about — not the value it returned.
type costRollupHarness struct {
	svc      *itemSvcImpl
	itemRepo *repositorymock.MockItemRepo
	written  []writtenUnitCost
}

type writtenUnitCost struct {
	Cost          decimal.Decimal
	DenominatorID string
}

func newCostRollupHarness(t *testing.T, stubs rolloutStubs) *costRollupHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &costRollupHarness{}

	step := &domain.ProductionFlowStep{
		ID: testCostStepID,
		Production: domain.StepProduction{
			ID:           "prod_1",
			ProducedItem: domain.LightItem{ID: testCostItemID, SKU: "04900"},
			Quantity: domain.BatchQuantity{
				ID:      "qty_prod",
				Measure: stubs.stepQuantity,
				Unit:    domain.LightUnit{ID: stubs.stepUnitID},
			},
		},
		LevelingFactor: "0",
		Allowances:     "0",
	}

	flowRepo := repositorymock.NewMockProductionFlowRepo(ctrl)
	flowRepo.EXPECT().FindStepsByProducedItem(gomock.Any(), testCostAccountID, testCostItemID).
		Return([]string{testCostStepID}, nil).AnyTimes()
	flowRepo.EXPECT().GetAllStepEdgesForAccount(gomock.Any(), testCostAccountID).
		Return(nil, nil).AnyTimes()
	flowRepo.EXPECT().GetFlowStep(gomock.Any(), testCostAccountID, testCostStepID).
		Return(step, nil).AnyTimes()

	h.itemRepo = repositorymock.NewMockItemRepo(ctrl)
	h.itemRepo.EXPECT().GetCostFlowConsumptions(gomock.Any(), testCostStepID).
		Return(stubs.consumptions, nil).AnyTimes()
	h.itemRepo.EXPECT().GetStockingUnit(gomock.Any(), testCostAccountID, testCostItemID).
		Return(&domain.ItemStockingUnit{UnitGroupID: testCostUnitGroupID, BaseUnitID: testCostUnitCarton}, nil).AnyTimes()
	h.itemRepo.EXPECT().UpdateUnitCost(gomock.Any(), testCostAccountID, testCostItemID, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, cost decimal.Decimal, denominatorUnitID string) *apierror.APIError {
			h.written = append(h.written, writtenUnitCost{Cost: cost, DenominatorID: denominatorUnitID})
			return nil
		}).AnyTimes()

	stepQueryRepo := repositorymock.NewMockProductionStepQueryRepo(ctrl)
	stepQueryRepo.EXPECT().Find(gomock.Any(), testCostAccountID, testCostStepID).
		Return(&domain.ProductionStepDetail{ID: testCostStepID}, nil).AnyTimes()

	unitRepo := repositorymock.NewMockUnitRepo(ctrl)
	unitRepo.EXPECT().IsUnitInGroup(gomock.Any(), testCostUnitGroupID, stubs.stepUnitID).
		Return(stubs.stepUnitInGroup, nil).AnyTimes()

	convRepo := repositorymock.NewMockUnitConversionRepo(ctrl)
	convRepo.EXPECT().ConvertValue(gomock.Any(), gomock.Any(), testCostUnitCarton, testCostUnitEach).
		Return(eachesPerCarton, nil).AnyTimes()

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewProductionFlowRepo().Return(flowRepo).AnyTimes()
	repos.EXPECT().NewItemRepo().Return(h.itemRepo).AnyTimes()
	repos.EXPECT().NewProductionStepQueryRepo().Return(stepQueryRepo).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(unitRepo).AnyTimes()
	repos.EXPECT().NewUnitConversionRepo().Return(convRepo).AnyTimes()

	h.svc = &itemSvcImpl{repos: repos}
	return h
}

// consumedCartonsAtPerEachCost is one carton drawn against a cost quoted per each — the consumption
// side and the cost side in different units, which the material term must not multiply together raw.
func consumedCartonsAtPerEachCost(cartons, costPerEach string) domain.CostFlowConsumption {
	return domain.CostFlowConsumption{
		ConsumedItemType:         "material",
		ConsumptionQuantity:      decimal.RequireFromString(cartons),
		ConsumptionUnitRatio:     eachesPerCarton,
		WasteQuantity:            decimal.Zero,
		WasteUnitRatio:           eachesPerCarton,
		UnitCost:                 decimal.RequireFromString(costPerEach),
		UnitCostDenominatorRatio: decimal.NewFromInt(1),
	}
}

// The rollup persists an item's cost against the unit the item is stocked in, whatever unit its
// production step is written in. Taking the denominator from the step instead is what let a
// per-carton figure be stored as a per-each one and value every carton eight times over.
func TestRecomputeItemCosts_WritesBackInTheItemsStockingUnit(t *testing.T) {
	t.Parallel()

	h := newCostRollupHarness(t, rolloutStubs{
		stepUnitID:      testCostUnitEach,
		stepQuantity:    decimal.NewFromInt(1),
		consumptions:    []domain.CostFlowConsumption{consumedCartonsAtPerEachCost("1", "2")},
		stepUnitInGroup: true,
	})

	costs, apiErr := h.svc.RecomputeItemCosts(context.Background(), testCostAccountID, testCostItemID)
	if apiErr != nil {
		t.Fatalf("RecomputeItemCosts: %v", apiErr)
	}

	if len(h.written) != 1 {
		t.Fatalf("expected one unit cost write, got %d", len(h.written))
	}
	// One carton of a $2/each material is $16 of material, and the step makes an each of it, so a
	// carton of the finished item costs eight of those.
	wantCost := decimal.RequireFromString("128")
	if !h.written[0].Cost.Equal(wantCost) {
		t.Errorf("persisted cost = %s, want %s", h.written[0].Cost, wantCost)
	}
	if h.written[0].DenominatorID != testCostUnitCarton {
		t.Errorf("persisted denominator = %s, want the item's stocking unit %s", h.written[0].DenominatorID, testCostUnitCarton)
	}
	if costs.UnitID != testCostUnitCarton {
		t.Errorf("reported unit = %s, want the item's stocking unit %s", costs.UnitID, testCostUnitCarton)
	}
	if costs.TotalCost != wantCost.StringFixed(30) {
		t.Errorf("reported total = %s, want %s", costs.TotalCost, wantCost.StringFixed(30))
	}
}

// GET /v1/catalog/items/{id}/costs recomputes and writes back on every call, so repeated page views
// are the load this runs under. Each pass has to land on the same value and the same denominator:
// a denominator that moves is a value that silently changes meaning, and the next pass compounds it.
func TestRecomputeItemCosts_RepeatedCallsAreIdempotent(t *testing.T) {
	t.Parallel()

	h := newCostRollupHarness(t, rolloutStubs{
		stepUnitID:      testCostUnitEach,
		stepQuantity:    decimal.NewFromInt(1),
		consumptions:    []domain.CostFlowConsumption{consumedCartonsAtPerEachCost("1", "2")},
		stepUnitInGroup: true,
	})

	for range 3 {
		if _, apiErr := h.svc.RecomputeItemCosts(context.Background(), testCostAccountID, testCostItemID); apiErr != nil {
			t.Fatalf("RecomputeItemCosts: %v", apiErr)
		}
	}

	if len(h.written) != 3 {
		t.Fatalf("expected three unit cost writes, got %d", len(h.written))
	}
	for i, got := range h.written {
		if !got.Cost.Equal(h.written[0].Cost) || got.DenominatorID != h.written[0].DenominatorID {
			t.Fatalf("call %d persisted %s/%s, want %s/%s — the write is not idempotent",
				i+1, got.Cost, got.DenominatorID, h.written[0].Cost, h.written[0].DenominatorID)
		}
	}
	if h.written[0].DenominatorID != testCostUnitCarton {
		t.Errorf("persisted denominator = %s, want the item's stocking unit %s", h.written[0].DenominatorID, testCostUnitCarton)
	}
}

// A step producing in a unit the item is not counted in has no conversion into the stocking unit, so
// there is no cost to write. Rejecting leaves the item's existing cost standing, which is wrong by
// however much its inputs have moved — a mislabelled one is wrong by a factor and spreads.
func TestRecomputeItemCosts_RejectsAStepUnitOutsideTheItemsUnitGroup(t *testing.T) {
	t.Parallel()

	h := newCostRollupHarness(t, rolloutStubs{
		stepUnitID:      testCostUnitEach,
		stepQuantity:    decimal.NewFromInt(1),
		stepUnitInGroup: false,
	})

	_, apiErr := h.svc.RecomputeItemCosts(context.Background(), testCostAccountID, testCostItemID)
	if apiErr == nil {
		t.Fatal("expected a validation error for a step unit outside the item's unit group")
	}
	if apiErr.Code != apierror.ErrorCodeValidationFailed {
		t.Errorf("error code = %v, want %v", apiErr.Code, apierror.ErrorCodeValidationFailed)
	}
	if len(h.written) != 0 {
		t.Errorf("expected no unit cost write, got %d", len(h.written))
	}
}

// The material term meets a quantity and the cost that prices it on base units. Multiplying them as
// entered reads a carton drawn against a per-each cost as one each's worth of money.
func TestCalculateStepCost_NormalisesConsumptionAgainstTheCostsOwnUnit(t *testing.T) {
	t.Parallel()

	step := &domain.ProductionFlowStep{
		Production:     domain.StepProduction{Quantity: domain.BatchQuantity{Measure: decimal.NewFromInt(1)}},
		LevelingFactor: "0",
		Allowances:     "0",
	}

	got := calculateStepCost(step, []domain.CostFlowConsumption{consumedCartonsAtPerEachCost("1", "2")})

	want := decimal.RequireFromString("16")
	if !got.material.Equal(want) {
		t.Errorf("material = %s, want %s", got.material, want)
	}
}

// A cost denominated in a unit outside the item's own group cannot be converted onto the item at all.
// ValidateCostRateUnits passes it, because a carton and an each are both counts and a dimension code
// cannot tell them apart.
func TestValidateCostDenominatorInUnitGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		inGroup  bool
		wantCode apierror.ErrorCode
	}{
		{"denominator in the item's unit group", true, ""},
		{"denominator outside the item's unit group", false, apierror.ErrorCodeValidationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := repositorymock.NewMockUnitRepo(ctrl)
			repo.EXPECT().IsUnitInGroup(gomock.Any(), testCostUnitGroupID, testCostUnitEach).
				Return(tt.inGroup, nil).AnyTimes()

			apiErr := ValidateCostDenominatorInUnitGroup(context.Background(), repo, testCostUnitGroupID, testCostUnitEach, "unit_cost")

			if tt.wantCode == "" {
				if apiErr != nil {
					t.Fatalf("expected no error, got %v", apiErr)
				}
				return
			}
			if apiErr == nil {
				t.Fatal("expected a validation error")
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("error code = %v, want %v", apiErr.Code, tt.wantCode)
			}
			if apiErr.Param != "unit_cost.denominator_unit_id" {
				t.Errorf("error param = %q, want unit_cost.denominator_unit_id", apiErr.Param)
			}
		})
	}
}
