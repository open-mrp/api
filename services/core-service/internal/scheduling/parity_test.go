package scheduling

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The parity gate: the Go solver must reproduce the TS script's plan for the same
// input. Fixtures are produced against real data by
//
//	cd dashboard/apps/api && KNIT_DUMP_JSON=1 KNIT_GROWTH_MULT=1 \
//	  bun run src/scripts/knit-scheduling-merz.ts
//
// then copied into testdata/ as merz_<date>_input.json and merz_<date>_expected.json.
//
// KNIT_GROWTH_MULT=1 matters: the Go port has no growth multiplier, so the fixture has
// to be un-multiplied demand. A blanket growth assumption is expressed here as a
// demand override instead.
//
// The test SKIPS when no fixture is present, because the dump requires production data
// that CI does not have. It must not be deleted for being skipped — it is the gate
// that runs before the solver is trusted with a real schedule.

type parityInput struct {
	PlanningAsOf string `json:"planningAsOf"`
	Settings     struct {
		HorizonWeeks                   int     `json:"horizonWeeks"`
		ShiftsPerDay                   int     `json:"shiftsPerDay"`
		HoursPerShift                  float64 `json:"hoursPerShift"`
		WorkDaysPerWeek                int     `json:"workDaysPerWeek"`
		WeeksPerYear                   int     `json:"weeksPerYear"`
		CapacityHeadroomPct            float64 `json:"capacityHeadroomPct"`
		DefaultLotUnits                float64 `json:"defaultLotUnits"`
		ChangeoverAvgMinutes           float64 `json:"changeoverAvgMinutes"`
		ChangeoverMinMinutes           float64 `json:"changeoverMinMinutes"`
		ChangeoverMaxMinutes           float64 `json:"changeoverMaxMinutes"`
		ChangeoverLaborRate            float64 `json:"changeoverLaborRate"`
		HoldingRatePct                 float64 `json:"holdingRatePct"`
		ServiceLevelZ                  float64 `json:"serviceLevelZ"`
		FinishLeadTimeWeeks            float64 `json:"finishLeadTimeWeeks"`
		DefaultConstraintLeadTimeWeeks float64 `json:"defaultConstraintLeadTimeWeeks"`
		MaxWeeksSupply                 float64 `json:"maxWeeksSupply"`
		GrowthMultiplier               float64 `json:"growthMultiplier"`
		DemandBasis                    string  `json:"demandBasis"`
	} `json:"settings"`
	Machines []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"machines"`
	Items []struct {
		ItemID                string   `json:"itemId"`
		SKU                   string   `json:"sku"`
		AnnualDemand          float64  `json:"annualDemand"`
		SecondsPerUnit        float64  `json:"secondsPerUnit"`
		UnitCost              float64  `json:"unitCost"`
		ChangeoverRate        float64  `json:"changeoverRate"`
		MeasuredLeadTimeWeeks float64  `json:"measuredLeadTimeWeeks"`
		SigmaWeeklyPooled     float64  `json:"sigmaWeeklyPooled"`
		SigmaDownstreamSum    float64  `json:"sigmaDownstreamSum"`
		OnHandEchelon         float64  `json:"onHandEchelon"`
		EligibleMachineIDs    []string `json:"eligibleMachineIds"`
		LotUnits              float64  `json:"lotUnits"`
	} `json:"items"`
}

type parityExpected struct {
	Policies []struct {
		SKU                     string  `json:"sku"`
		WeeklyDemand            float64 `json:"weeklyDemand"`
		EOQUnits                float64 `json:"eoqUnits"`
		SetupCost               float64 `json:"setupCost"`
		HoldingCost             float64 `json:"holdingCost"`
		ConstraintLeadTimeWeeks float64 `json:"constraintLeadTimeWeeks"`
		SafetyStockPrimary      float64 `json:"safetyStockPrimary"`
		SafetyStockDownstream   float64 `json:"safetyStockDownstream"`
		ReorderPoint            float64 `json:"reorderPoint"`
		ABCClass                string  `json:"abcClass"`
	} `json:"policies"`
	Campaigns []struct {
		SKU       string  `json:"sku"`
		WeekIndex int     `json:"weekIndex"`
		Units     float64 `json:"units"`
		MachineID string  `json:"machineId"`
	} `json:"campaigns"`
}

// relClose compares within a relative tolerance, falling back to absolute near zero.
func relClose(got, want, tol float64) bool {
	if math.Abs(want) < 1e-9 {
		return math.Abs(got) < tol
	}
	return math.Abs(got-want)/math.Abs(want) < tol
}

func loadParityFixture(t *testing.T) (*parityInput, *parityExpected) {
	t.Helper()

	inputs, err := filepath.Glob(filepath.Join("testdata", "*_input.json"))
	if err != nil || len(inputs) == 0 {
		t.Skip("no parity fixture in testdata/ — see the comment at the top of this file")
	}

	inputPath := inputs[0]
	expectedPath := inputPath[:len(inputPath)-len("_input.json")] + "_expected.json"

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read %s: %v", inputPath, err)
	}
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read %s: %v", expectedPath, err)
	}

	var in parityInput
	if err := json.Unmarshal(inputBytes, &in); err != nil {
		t.Fatalf("parse %s: %v", inputPath, err)
	}
	var want parityExpected
	if err := json.Unmarshal(expectedBytes, &want); err != nil {
		t.Fatalf("parse %s: %v", expectedPath, err)
	}

	// A fixture captured with the growth multiplier still on would compare the Go
	// solver against doubled demand and fail for a reason that has nothing to do with
	// the port.
	if in.Settings.GrowthMultiplier != 1 {
		t.Fatalf("fixture was captured with growthMultiplier=%v; recapture with KNIT_GROWTH_MULT=1",
			in.Settings.GrowthMultiplier)
	}

	return &in, &want
}

func settingsFromFixture(in *parityInput) Settings {
	s := DefaultSettings()
	s.HorizonWeeks = in.Settings.HorizonWeeks
	s.ShiftsPerDay = in.Settings.ShiftsPerDay
	s.HoursPerShift = in.Settings.HoursPerShift
	s.WorkDaysPerWeek = in.Settings.WorkDaysPerWeek
	s.WeeksPerYear = in.Settings.WeeksPerYear
	s.CapacityHeadroomPct = in.Settings.CapacityHeadroomPct
	s.DefaultLotUnits = in.Settings.DefaultLotUnits
	s.ChangeoverAvgMinutes = in.Settings.ChangeoverAvgMinutes
	s.ChangeoverMinMinutes = in.Settings.ChangeoverMinMinutes
	s.ChangeoverMaxMinutes = in.Settings.ChangeoverMaxMinutes
	s.ChangeoverLaborRate = in.Settings.ChangeoverLaborRate
	s.HoldingRatePct = in.Settings.HoldingRatePct
	s.ServiceLevelZ = in.Settings.ServiceLevelZ
	s.FinishLeadTimeWeeks = in.Settings.FinishLeadTimeWeeks
	s.DefaultConstraintLeadTimeWeeks = in.Settings.DefaultConstraintLeadTimeWeeks
	s.MaxWeeksSupply = in.Settings.MaxWeeksSupply
	return s
}

func TestParity_PolicyMatchesScript(t *testing.T) {
	in, want := loadParityFixture(t)
	s := settingsFromFixture(in)

	policies := make(map[string]ItemPolicy, len(in.Items))
	for _, item := range in.Items {
		p := ComputePolicy(PolicyInput{
			ItemID:         item.ItemID,
			SKU:            item.SKU,
			AnnualDemand:   item.AnnualDemand,
			SecondsPerUnit: item.SecondsPerUnit,
			UnitCost:       item.UnitCost,
			// The script stores a single combined coRate (tech rate + step overhead);
			// the Go model keeps the tech rate in settings and the overhead per item,
			// so recover the overhead by subtracting the tech rate back out.
			OverheadRate:          item.ChangeoverRate - s.ChangeoverLaborRate,
			MeasuredLeadTimeWeeks: item.MeasuredLeadTimeWeeks,
			SigmaWeeklyPooled:     item.SigmaWeeklyPooled,
			SigmaDownstreamSum:    item.SigmaDownstreamSum,
			OnHandEchelon:         item.OnHandEchelon,
		}, s)
		policies[item.SKU] = p
	}

	const tol = 1e-6
	for _, expected := range want.Policies {
		got, ok := policies[expected.SKU]
		if !ok {
			t.Errorf("%s: missing from Go policy output", expected.SKU)
			continue
		}

		checks := []struct {
			name      string
			got, want float64
		}{
			{"weeklyDemand", got.WeeklyDemand, expected.WeeklyDemand},
			{"eoqUnits", got.EOQUnits, expected.EOQUnits},
			{"setupCost", got.SetupCost, expected.SetupCost},
			{"holdingCost", got.HoldingCost, expected.HoldingCost},
			{"safetyStockPrimary", got.SafetyStockPrimary, expected.SafetyStockPrimary},
			{"safetyStockDownstream", got.SafetyStockDownstream, expected.SafetyStockDownstream},
			{"reorderPoint", got.ReorderPoint, expected.ReorderPoint},
		}
		for _, c := range checks {
			if !relClose(c.got, c.want, tol) {
				t.Errorf("%s %s: got %v, want %v", expected.SKU, c.name, c.got, c.want)
			}
		}
	}
}

func TestParity_PlanMatchesScript(t *testing.T) {
	in, want := loadParityFixture(t)
	s := settingsFromFixture(in)

	items := make([]LevellingItem, 0, len(in.Items))
	for _, item := range in.Items {
		eligible := make(map[string]bool, len(item.EligibleMachineIDs))
		for _, id := range item.EligibleMachineIDs {
			eligible[id] = true
		}

		lotUnits := item.LotUnits
		if lotUnits <= 0 {
			lotUnits = s.DefaultLotUnits
		}

		items = append(items, LevellingItem{
			Policy: ComputePolicy(PolicyInput{
				ItemID:         item.ItemID,
				SKU:            item.SKU,
				AnnualDemand:   item.AnnualDemand,
				SecondsPerUnit: item.SecondsPerUnit,
				UnitCost:       item.UnitCost,
				// The script stores a single combined coRate (tech rate + step overhead);
				// the Go model keeps the tech rate in settings and the overhead per item,
				// so recover the overhead by subtracting the tech rate back out.
				OverheadRate:          item.ChangeoverRate - s.ChangeoverLaborRate,
				MeasuredLeadTimeWeeks: item.MeasuredLeadTimeWeeks,
				SigmaWeeklyPooled:     item.SigmaWeeklyPooled,
				SigmaDownstreamSum:    item.SigmaDownstreamSum,
				OnHandEchelon:         item.OnHandEchelon,
			}, s),
			EligibleMachineID: eligible,
			LotUnits:          lotUnits,
		})
	}

	machines := make([]Machine, len(in.Machines))
	for i, m := range in.Machines {
		machines[i] = Machine{ID: m.ID, Name: m.Name}
	}

	got := Level(items, machines, s, nil)

	type key struct {
		sku  string
		week int
	}
	gotByKey := make(map[key]Campaign, len(got.Campaigns))
	for _, c := range got.Campaigns {
		gotByKey[key{c.SKU, c.WeekIndex}] = c
	}

	if len(got.Campaigns) != len(want.Campaigns) {
		t.Errorf("campaign count: got %d, want %d", len(got.Campaigns), len(want.Campaigns))
	}

	for _, expected := range want.Campaigns {
		k := key{expected.SKU, expected.WeekIndex}
		gotCampaign, ok := gotByKey[k]
		if !ok {
			t.Errorf("%s week %d: script planned %v units, Go planned nothing",
				expected.SKU, expected.WeekIndex, expected.Units)
			continue
		}
		if !relClose(gotCampaign.Units, expected.Units, 1e-6) {
			t.Errorf("%s week %d units: got %v, want %v",
				expected.SKU, expected.WeekIndex, gotCampaign.Units, expected.Units)
		}
		if expected.MachineID != "" && gotCampaign.MachineID != expected.MachineID {
			t.Errorf("%s week %d machine: got %q, want %q",
				expected.SKU, expected.WeekIndex, gotCampaign.MachineID, expected.MachineID)
		}
		delete(gotByKey, k)
	}

	for k, c := range gotByKey {
		t.Errorf("%s week %d: Go planned %v units, script planned nothing", k.sku, k.week, c.Units)
	}
}
