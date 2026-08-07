package scheduling

import "testing"

func TestResolveFulfillmentPolicy_ItemOverrideWinsTheChain(t *testing.T) {
	t.Parallel()

	got := ResolveFulfillmentPolicy("itm_a", PolicyResolutionInput{
		ItemOverrides:       map[string]string{"itm_a": PolicyMakeToOrder},
		ProductLineByItem:   map[string]string{"itm_a": "pdln_1"},
		PolicyByProductLine: map[string]string{"pdln_1": PolicyMakeToStock},
		AccountDefault:      PolicyMakeToStock,
	})
	if got.Policy != PolicyMakeToOrder || got.Source != PolicySourceItem {
		t.Fatalf("got %+v, want make_to_order/item", got)
	}
}

func TestResolveFulfillmentPolicy_FallsThroughLineThenAccount(t *testing.T) {
	t.Parallel()

	line := ResolveFulfillmentPolicy("itm_a", PolicyResolutionInput{
		ProductLineByItem:   map[string]string{"itm_a": "pdln_1"},
		PolicyByProductLine: map[string]string{"pdln_1": PolicyMakeToOrder},
		AccountDefault:      PolicyMakeToStock,
	})
	if line.Policy != PolicyMakeToOrder || line.Source != PolicySourceProductLine {
		t.Fatalf("got %+v, want make_to_order/product_line", line)
	}

	account := ResolveFulfillmentPolicy("itm_a", PolicyResolutionInput{
		AccountDefault: PolicyMakeToOrder,
	})
	if account.Policy != PolicyMakeToOrder || account.Source != PolicySourceAccountDefault {
		t.Fatalf("got %+v, want make_to_order/account_default", account)
	}
}

// Nothing configured anywhere must resolve to make-to-stock, which is how every plan behaved before policies existed.
func TestResolveFulfillmentPolicy_DefaultsToMakeToStock(t *testing.T) {
	t.Parallel()

	got := ResolveFulfillmentPolicy("itm_a", PolicyResolutionInput{})
	if got.Policy != PolicyMakeToStock || got.Source != PolicySourceAccountDefault {
		t.Fatalf("got %+v, want make_to_stock/account_default", got)
	}
}

// Greige is not sold and has no line of its own, so it inherits from what it becomes — but only when the whole family agrees.
func TestResolveFulfillmentPolicy_IntermediateInheritsOnlyWhenEveryDescendantAgrees(t *testing.T) {
	t.Parallel()

	family := []FinishedGood{
		{ItemID: "itm_fg1", ProductLineID: "pdln_socks"},
		{ItemID: "itm_fg2", ProductLineID: "pdln_socks"},
	}

	// The whole family is make-to-order, so the greige behind it is too.
	all := ResolveFulfillmentPolicy("itm_greige", PolicyResolutionInput{
		PolicyByProductLine: map[string]string{"pdln_socks": PolicyMakeToOrder},
		AccountDefault:      PolicyMakeToStock,
		DownstreamByItem:    map[string][]FinishedGood{"itm_greige": family},
	})
	if all.Policy != PolicyMakeToOrder {
		t.Fatalf("a wholly make-to-order family should make its greige make-to-order, got %+v", all)
	}

	// One stocked sibling means the greige still has to hold a buffer for it.
	mixed := ResolveFulfillmentPolicy("itm_greige", PolicyResolutionInput{
		ItemOverrides:       map[string]string{"itm_fg1": PolicyMakeToOrder},
		PolicyByProductLine: map[string]string{"pdln_socks": PolicyMakeToStock},
		AccountDefault:      PolicyMakeToStock,
		DownstreamByItem:    map[string][]FinishedGood{"itm_greige": family},
	})
	if mixed.Policy != PolicyMakeToStock {
		t.Fatalf("one stocked sibling must keep the greige make-to-stock, got %+v", mixed)
	}
}

// A make-to-order item holds no statistical buffer: it is not built until the demand exists, so there is nothing to buffer against.
func TestComputePolicy_MakeToOrderHoldsNoBuffer(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	in := PolicyInput{
		ItemID:                "itm_a",
		SKU:                   "SKU-A",
		AnnualDemand:          5200,
		SecondsPerUnit:        10,
		UnitCost:              4,
		MeasuredLeadTimeWeeks: 2,
		SigmaWeeklyPooled:     40,
		SigmaDownstreamSum:    60,
		OnHandEchelon:         100,
	}

	stock := ComputePolicy(in, s)
	in.FulfillmentPolicy = PolicyMakeToOrder
	order := ComputePolicy(in, s)

	if stock.SafetyStockPrimary <= 0 || stock.ReorderPoint <= 0 || stock.OrderUpTo <= 0 {
		t.Fatalf("precondition: make-to-stock should hold a buffer, got %+v", stock)
	}
	if order.SafetyStockPrimary != 0 || order.SafetyStockDownstream != 0 {
		t.Fatalf("make-to-order should hold no safety stock, got primary=%v downstream=%v",
			order.SafetyStockPrimary, order.SafetyStockDownstream)
	}
	if order.ReorderPoint != 0 || order.OrderUpTo != 0 {
		t.Fatalf("make-to-order should have no statistical trigger or ceiling, got rop=%v upTo=%v",
			order.ReorderPoint, order.OrderUpTo)
	}
	// EOQ is still computed: it describes the item, and reporting keeps it even where the sweep does not size campaigns by it.
	if order.EOQUnits <= 0 {
		t.Fatal("EOQ should still be reported for a make-to-order item")
	}
}

// An unset policy has to behave exactly like make-to-stock, or adopting the field would change every existing plan.
func TestComputePolicy_UnsetPolicyIsMakeToStock(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	in := PolicyInput{
		ItemID: "itm_a", SKU: "SKU-A", AnnualDemand: 5200, SecondsPerUnit: 10, UnitCost: 4,
		MeasuredLeadTimeWeeks: 2, SigmaWeeklyPooled: 40, SigmaDownstreamSum: 60, OnHandEchelon: 100,
	}

	unset := ComputePolicy(in, s)
	in.FulfillmentPolicy = PolicyMakeToStock
	explicit := ComputePolicy(in, s)

	if unset != explicit {
		t.Fatalf("an unset policy must equal make-to-stock exactly:\n unset=%+v\n explicit=%+v", unset, explicit)
	}
	if unset.FulfillmentPolicy != PolicyMakeToStock {
		t.Fatalf("policy should be normalized to make_to_stock, got %q", unset.FulfillmentPolicy)
	}
}

// A make-to-order item is triggered by the order book over its lead time, not by an average.
func TestLevellingItem_MakeToOrderTriggerIsTheDatedOrderBook(t *testing.T) {
	t.Parallel()

	item := LevellingItem{
		Policy: ItemPolicy{
			FulfillmentPolicy:       PolicyMakeToOrder,
			ConstraintLeadTimeWeeks: 1,
			FinishLeadTimeWeeks:     1,
		},
		// Lead time is 2 weeks, so week 0 looks through week 2 inclusive.
		FirmByWeek: []float64{0, 100, 250, 500, 900},
	}

	if got := item.triggerForWeek(0); got != 350 {
		t.Fatalf("week 0 trigger = %v, want 350 (weeks 0..2)", got)
	}
	if got := item.triggerForWeek(2); got != 1650 {
		t.Fatalf("week 2 trigger = %v, want 1650 (weeks 2..4)", got)
	}
	// Nothing on the book in range means nothing to build.
	empty := LevellingItem{Policy: ItemPolicy{FulfillmentPolicy: PolicyMakeToOrder}}
	if got := empty.triggerForWeek(0); got != 0 {
		t.Fatalf("an empty order book should trigger nothing, got %v", got)
	}
}

// Make-to-stock keeps the constant (s,S) trigger it always had.
func TestLevellingItem_MakeToStockTriggerIsUnchanged(t *testing.T) {
	t.Parallel()

	item := LevellingItem{Policy: ItemPolicy{ReorderPoint: 500, OrderUpTo: 300}}
	for week := range 5 {
		if got := item.triggerForWeek(week); got != 300 {
			t.Fatalf("week %d = %v, want the lower of ROP and order-up-to (300)", week, got)
		}
	}
}

// The end-to-end shape of the policy: a make-to-order item with nothing on the book is never built, and the same item with an order is built once, to the size of the order.
func TestLevel_MakeToOrderBuildsOnlyWhatIsOrdered(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	s.HorizonWeeks = 8

	base := LevellingItem{
		Policy: ItemPolicy{
			ItemID: "itm_a", SKU: "SKU-A",
			FulfillmentPolicy:       PolicyMakeToOrder,
			SecondsPerUnit:          10,
			ConstraintLeadTimeWeeks: 1,
			FinishLeadTimeWeeks:     1,
			EOQUnits:                10000, // deliberately huge: it must not be what sizes the campaign
		},
		LotUnits: 60,
	}
	machines := []Machine{{ID: "mc_1", Name: "1"}}

	idle := Level([]LevellingItem{base}, machines, s, nil)
	if len(idle.Campaigns) != 0 {
		t.Fatalf("a make-to-order item with an empty order book must not be built, got %d campaigns", len(idle.Campaigns))
	}

	ordered := base
	ordered.FirmByWeek = make([]float64, s.HorizonWeeks)
	ordered.FirmByWeek[3] = 300

	got := Level([]LevellingItem{ordered}, machines, s, nil)
	if len(got.Campaigns) == 0 {
		t.Fatal("an order on the book must produce a campaign")
	}
	var total float64
	for _, c := range got.Campaigns {
		total += c.Units
	}
	// 300 rounded up to whole 60-unit lots is exactly 300; the huge EOQ must not inflate it.
	if total != 300 {
		t.Fatalf("built %v units for a 300-unit order, want 300 (EOQ must not size a make-to-order campaign)", total)
	}
}

// A promise outranks a buffer when they contend for the same machine-hour.
func TestLevel_MakeToOrderIsServedBeforeMakeToStock(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	s.HorizonWeeks = 1
	s.ShiftsPerDay = 1
	s.HoursPerShift = 1
	s.WorkDaysPerWeek = 1
	s.CapacityHeadroomPct = 1

	// One machine-hour, and each item wants all of it, so only one can be served.
	mto := LevellingItem{
		Policy: ItemPolicy{
			ItemID: "itm_mto", SKU: "AAA-first-alphabetically",
			FulfillmentPolicy: PolicyMakeToOrder,
			SecondsPerUnit:    60,
		},
		LotUnits:   60,
		FirmByWeek: []float64{60},
	}
	mts := LevellingItem{
		Policy: ItemPolicy{
			ItemID: "itm_mts", SKU: "ZZZ-last-alphabetically",
			FulfillmentPolicy: PolicyMakeToStock,
			SecondsPerUnit:    60,
			WeeklyDemand:      60,
			ReorderPoint:      100000, // desperately short, so gap-to-ROP would rank it first
			OrderUpTo:         100000,
			EOQUnits:          60,
		},
		LotUnits: 60,
	}

	got := Level([]LevellingItem{mts, mto}, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)
	if len(got.Campaigns) != 1 {
		t.Fatalf("only one campaign fits the hour, got %d", len(got.Campaigns))
	}
	if got.Campaigns[0].ItemID != "itm_mto" {
		t.Fatalf("the committed order should win the machine-hour, got %s", got.Campaigns[0].ItemID)
	}
}
