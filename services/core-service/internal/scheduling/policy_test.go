package scheduling

import (
	"math"
	"testing"
)

// The greige stage holds its buffer plus whatever a campaign leaves on top of it. These
// are the numbers that answer "how big does the greige store need to be", which the
// echelon total cannot be decomposed back into once summed.
func TestComputePolicy_GreigeStageHoldingIsBufferPlusCampaign(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	p := ComputePolicy(PolicyInput{
		ItemID:            "it_greige",
		SKU:               "GREIGE-1",
		AnnualDemand:      52000,
		SecondsPerUnit:    30,
		UnitCost:          4,
		SigmaWeeklyPooled: 200,
		OnHandEchelon:     9000,
		OnHandGreige:      1500,
	}, s)

	if p.OnHandGreige != 1500 {
		t.Errorf("greige on hand = %v, want 1500 kept separate from the echelon total", p.OnHandGreige)
	}
	if p.OnHandEchelon != 9000 {
		t.Errorf("echelon on hand = %v, want 9000; the build decision still uses the pooled figure", p.OnHandEchelon)
	}

	wantAvg := p.SafetyStockPrimary + p.EOQUnits/2
	if math.Abs(p.AverageGreigeInventory-wantAvg) > 1e-9 {
		t.Errorf("average greige inventory = %v, want %v (buffer + half a campaign)", p.AverageGreigeInventory, wantAvg)
	}
	wantMax := p.SafetyStockPrimary + p.EOQUnits
	if math.Abs(p.MaxGreigeInventory-wantMax) > 1e-9 {
		t.Errorf("max greige inventory = %v, want %v (buffer + a whole campaign)", p.MaxGreigeInventory, wantMax)
	}
}

// Each finished SKU answers for its own demand against its own stock. Sizing them off a
// share of the pooled figure would make a slow mover look as urgent as a fast one.
func TestComputeFinishedPolicies_PerSKUTargetsFromOwnDemand(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	greige := ComputePolicy(PolicyInput{ItemID: "it_greige", SKU: "GREIGE-1", AnnualDemand: 52000, SecondsPerUnit: 30, UnitCost: 4}, s)

	got := ComputeFinishedPolicies(greige, []FinishedGoodDemand{
		{ItemID: "it_fast", SKU: "FG-FAST", AnnualDemand: 41600, SigmaWeekly: 150, OnHand: 3000},
		{ItemID: "it_slow", SKU: "FG-SLOW", AnnualDemand: 5200, SigmaWeekly: 20, OnHand: 100},
	}, s)

	if len(got) != 2 {
		t.Fatalf("got %d finished policies, want 2", len(got))
	}
	// Sorted by SKU, so FAST comes first.
	fast, slow := got[0], got[1]
	if fast.SKU != "FG-FAST" || slow.SKU != "FG-SLOW" {
		t.Fatalf("order = %s, %s; want a stable sort by SKU", fast.SKU, slow.SKU)
	}

	for _, p := range got {
		if p.GreigeItemID != "it_greige" || p.GreigeSKU != "GREIGE-1" {
			t.Errorf("%s lost its greige parent: %q/%q", p.SKU, p.GreigeItemID, p.GreigeSKU)
		}
	}

	// The buffer covers the finishing lead time only. Charging the knit lead time here
	// too would double-count the greige buffer sitting behind this stock.
	wantSS := s.ServiceLevelZ * 150 * math.Sqrt(s.FinishLeadTimeWeeks)
	if math.Abs(fast.SafetyStock-wantSS) > 1e-9 {
		t.Errorf("safety stock = %v, want %v (z * sigma * sqrt(finish LT))", fast.SafetyStock, wantSS)
	}
	wantROP := fast.WeeklyDemand*s.FinishLeadTimeWeeks + wantSS
	if math.Abs(fast.ReorderPoint-wantROP) > 1e-9 {
		t.Errorf("reorder point = %v, want %v (finishing lead time only)", fast.ReorderPoint, wantROP)
	}

	if fast.WeeklyDemand <= slow.WeeklyDemand {
		t.Error("the fast mover must carry the larger weekly demand; targets are per SKU, not a share of the pool")
	}
	if fast.OnHand != 3000 || slow.OnHand != 100 {
		t.Error("each finished SKU must be measured against its own stock, not the family's")
	}
}

// The two stages must not overlap: greige holds its own buffer, finished goods hold
// theirs, and adding them is the network total rather than a double count.
func TestFinishedAndGreigeStagesDoNotDoubleCount(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	greige := ComputePolicy(PolicyInput{
		ItemID: "it_greige", SKU: "GREIGE-1",
		AnnualDemand: 52000, SecondsPerUnit: 30, UnitCost: 4,
		SigmaWeeklyPooled: 200, SigmaDownstreamSum: 260,
	}, s)

	finished := ComputeFinishedPolicies(greige, []FinishedGoodDemand{
		{ItemID: "it_a", SKU: "FG-A", AnnualDemand: 26000, SigmaWeekly: 130},
		{ItemID: "it_b", SKU: "FG-B", AnnualDemand: 26000, SigmaWeekly: 130},
	}, s)

	var finishedSafety float64
	for _, p := range finished {
		finishedSafety += p.SafetyStock
	}

	// The greige policy's downstream figure is the same money, counted once at the
	// family level; the per-SKU rows are its decomposition, not an addition to it.
	if math.Abs(finishedSafety-greige.SafetyStockDownstream) > 1e-6 {
		t.Errorf("finished safety stock sums to %v but the greige policy reports %v; the decomposition must not change the total",
			finishedSafety, greige.SafetyStockDownstream)
	}

	// And the greige buffer is a different number entirely — pooled, so strictly less
	// than the plain sum it decomposes into.
	if greige.SafetyStockPrimary >= finishedSafety {
		t.Errorf("pooled greige buffer %v should be smaller than the summed finished buffers %v; pooling is the reason the buffer sits at greige",
			greige.SafetyStockPrimary, finishedSafety)
	}
}
