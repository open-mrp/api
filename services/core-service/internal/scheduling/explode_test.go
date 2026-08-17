package scheduling

import (
	"reflect"
	"testing"
	"time"
)

// knitSewPackFlow is a straight three-step line: knit feeds sew, sew feeds pack.
func knitSewPackFlow() ExplosionInput {
	return ExplosionInput{
		Campaigns: []ExplosionCampaign{{
			LineID: "pnscln_1", ItemID: "it_knit", SKU: "KNIT-1",
			MachineID: "mc_1", WeekIndex: 0, Quantity: 1000, StepID: "prs_knit",
		}},
		Edges: []StepEdge{
			{UpstreamStepID: "prs_knit", DownstreamStepID: "prs_sew"},
			{UpstreamStepID: "prs_sew", DownstreamStepID: "prs_pack"},
		},
		Steps: map[string]StepInfo{
			"prs_knit": {StepID: "prs_knit", DepartmentID: "dp_knit", Name: "Knit", YieldRatio: 1},
			"prs_sew":  {StepID: "prs_sew", DepartmentID: "dp_sew", Name: "Sew", LeadTimeOffsetWeeks: 1, YieldRatio: 1},
			"prs_pack": {StepID: "prs_pack", DepartmentID: "dp_pack", Name: "Pack", LeadTimeOffsetWeeks: 2, YieldRatio: 1},
		},
	}
}

func TestExplode_WalksDownstreamAndAccumulatesOffsets(t *testing.T) {
	got := Explode(knitSewPackFlow())

	if len(got) != 3 {
		t.Fatalf("expected 3 derived lines (knit, sew, pack), got %d: %+v", len(got), got)
	}

	knit, sew, pack := got[0], got[1], got[2]

	if knit.ProductionStep != "prs_knit" || knit.Depth != 0 || knit.WeekIndex != 0 {
		t.Errorf("first derived line = %+v, want the constraint step itself at depth 0 in its own week", knit)
	}

	if sew.ProductionStep != "prs_sew" || sew.DepartmentID != "dp_sew" {
		t.Errorf("second derived line = %+v, want the sew step", sew)
	}
	if sew.WeekIndex != 1 {
		t.Errorf("sew week = %d, want 1 — one week after the constraint campaign", sew.WeekIndex)
	}
	if sew.Depth != 1 {
		t.Errorf("sew depth = %d, want 1", sew.Depth)
	}

	// Offsets accumulate along the path: pack is 2 weeks after sew, not after knit.
	if pack.WeekIndex != 3 {
		t.Errorf("pack week = %d, want 3 — offsets accumulate down the chain", pack.WeekIndex)
	}
	if pack.Depth != 2 {
		t.Errorf("pack depth = %d, want 2", pack.Depth)
	}
}

func TestExplode_AppliesYieldLossDownstream(t *testing.T) {
	in := knitSewPackFlow()
	sew := in.Steps["prs_sew"]
	sew.YieldRatio = 0.9
	in.Steps["prs_sew"] = sew

	got := Explode(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 derived lines, got %d", len(got))
	}

	// The constraint step is what the plan already committed to, so no yield applies to it.
	if got[0].Quantity != 1000 {
		t.Errorf("knit quantity = %v, want 1000 — the constraint campaign is the plan's own figure", got[0].Quantity)
	}
	if got[1].Quantity != 900 {
		t.Errorf("sew quantity = %v, want 900 — a 10%% loss at sew", got[1].Quantity)
	}
	// The loss compounds: pack sees what sew actually produced, not what knit started with.
	if got[2].Quantity != 900 {
		t.Errorf("pack quantity = %v, want 900 — downstream inherits the upstream loss", got[2].Quantity)
	}
}

// A plant whose constraint has nothing configured downstream of it still has scheduled work, and the work list is the surface that has to show it.
func TestExplode_ConstraintOnlyFlowStillProducesWork(t *testing.T) {
	got := Explode(ExplosionInput{
		Campaigns: []ExplosionCampaign{{
			LineID: "pnscln_1", ItemID: "it_knit", SKU: "KNIT-1",
			MachineID: "mc_51", WeekIndex: 2, Quantity: 600, StepID: "prs_knit",
		}},
		Steps: map[string]StepInfo{
			"prs_knit": {StepID: "prs_knit", DepartmentID: "dp_knit", Name: "Knit", YieldRatio: 1},
		},
	})

	if len(got) != 1 {
		t.Fatalf("expected the constraint campaign itself, got %d lines: %+v", len(got), got)
	}
	if got[0].WeekIndex != 2 || got[0].Quantity != 600 || got[0].DepartmentID != "dp_knit" {
		t.Errorf("derived line = %+v, want the campaign as-planned in its own department", got[0])
	}
}

func TestExplode_ZeroYieldTreatedAsNoLoss(t *testing.T) {
	// An unconfigured yield is missing data, not a step that destroys everything.
	in := knitSewPackFlow()
	sew := in.Steps["prs_sew"]
	sew.YieldRatio = 0
	in.Steps["prs_sew"] = sew

	got := Explode(in)
	if got[1].Quantity != 1000 {
		t.Errorf("sew quantity = %v, want 1000 — an unset yield must not zero the plan", got[1].Quantity)
	}
}

// A quality step feeding back into the step before it is a real production pattern. An
// unbounded walk over it does not return.
func TestExplode_TerminatesOnReworkLoop(t *testing.T) {
	in := ExplosionInput{
		Campaigns: []ExplosionCampaign{{
			LineID: "pnscln_1", ItemID: "it_1", SKU: "SKU-1",
			WeekIndex: 0, Quantity: 100, StepID: "prs_a",
		}},
		Edges: []StepEdge{
			{UpstreamStepID: "prs_a", DownstreamStepID: "prs_b"},
			{UpstreamStepID: "prs_b", DownstreamStepID: "prs_a"},
		},
		Steps: map[string]StepInfo{
			"prs_a": {StepID: "prs_a", DepartmentID: "dp_a", YieldRatio: 1},
			"prs_b": {StepID: "prs_b", DepartmentID: "dp_b", YieldRatio: 1},
		},
		MaxDepth: 4,
	}

	done := make(chan []DerivedLine, 1)
	go func() { done <- Explode(in) }()

	select {
	case got := <-done:
		if len(got) == 0 {
			t.Fatal("a rework loop should still produce derived work, not nothing")
		}
		for _, line := range got {
			if line.Depth > in.MaxDepth {
				t.Errorf("derived line at depth %d exceeds the bound of %d", line.Depth, in.MaxDepth)
			}
		}
	case <-time.After(5 * time.Second):
		// Explode is synchronous and fast; five seconds means it is looping.
		t.Fatal("Explode did not terminate on a rework loop")
	}
}

func TestExplode_RespectsDepthBound(t *testing.T) {
	in := ExplosionInput{
		Campaigns: []ExplosionCampaign{{
			LineID: "l", ItemID: "i", SKU: "S", WeekIndex: 0, Quantity: 10, StepID: "s0",
		}},
		MaxDepth: 2,
		Steps:    map[string]StepInfo{},
	}
	// A long chain: s0 → s1 → … → s5.
	for i := 0; i < 5; i++ {
		up := "s" + string(rune('0'+i))
		down := "s" + string(rune('0'+i+1))
		in.Edges = append(in.Edges, StepEdge{UpstreamStepID: up, DownstreamStepID: down})
		in.Steps[down] = StepInfo{StepID: down, DepartmentID: "dp" + down, YieldRatio: 1}
	}

	got := Explode(in)
	for _, line := range got {
		if line.Depth > 2 {
			t.Errorf("derived line at depth %d exceeds the bound of 2", line.Depth)
		}
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 derived lines within the bound, got %d", len(got))
	}
}

// Go randomizes map iteration, so without explicit sorting two runs over the same plan
// produce the same rows in a different order and version diffs become unreadable.
func TestExplode_Deterministic(t *testing.T) {
	in := ExplosionInput{
		Campaigns: []ExplosionCampaign{
			{LineID: "l2", ItemID: "i2", SKU: "BBB", WeekIndex: 1, Quantity: 50, StepID: "prs_knit"},
			{LineID: "l1", ItemID: "i1", SKU: "AAA", WeekIndex: 0, Quantity: 100, StepID: "prs_knit"},
		},
		Edges: []StepEdge{
			{UpstreamStepID: "prs_knit", DownstreamStepID: "prs_sew"},
			{UpstreamStepID: "prs_knit", DownstreamStepID: "prs_wash"},
			{UpstreamStepID: "prs_knit", DownstreamStepID: "prs_dye"},
		},
		Steps: map[string]StepInfo{
			"prs_sew":  {StepID: "prs_sew", DepartmentID: "dp_sew", YieldRatio: 1},
			"prs_wash": {StepID: "prs_wash", DepartmentID: "dp_wash", YieldRatio: 1},
			"prs_dye":  {StepID: "prs_dye", DepartmentID: "dp_dye", YieldRatio: 1},
		},
	}

	first := Explode(in)
	for i := 0; i < 50; i++ {
		if got := Explode(in); !reflect.DeepEqual(first, got) {
			t.Fatalf("run %d differed from the first run:\nfirst: %+v\ngot:   %+v", i, first, got)
		}
	}

	// Campaigns are ordered by week then SKU, so the earlier week comes first
	// regardless of the order they were supplied in.
	if first[0].SKU != "AAA" {
		t.Errorf("first derived line SKU = %q, want AAA — campaigns sort by week then SKU", first[0].SKU)
	}
}

func TestExplode_SkipsStepsWithNoMetadata(t *testing.T) {
	in := knitSewPackFlow()
	delete(in.Steps, "prs_sew")

	got := Explode(in)

	// Sew has no department, so it cannot be scheduled; pack is only reachable through
	// it, so the chain stops rather than attributing work to the wrong department.
	for _, line := range got {
		if line.ProductionStep == "prs_sew" {
			t.Errorf("a step with no metadata must not produce derived work: %+v", line)
		}
	}
}

func TestExplode_NoCampaignsIsEmpty(t *testing.T) {
	if got := Explode(ExplosionInput{}); len(got) != 0 {
		t.Errorf("expected no derived work from no campaigns, got %+v", got)
	}
}
